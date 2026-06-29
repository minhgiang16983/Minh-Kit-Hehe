package ratelimit

import (
	"context"
	"fmt"
	"strings"

	"sync"

	"github.com/minhgiang16983/Minh-Kit-Hehe/attributes"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// MultiLevelRule defines a multi-level rate limiting rule
type MultiLevelRule struct {
	Name         string           `json:"name" yaml:"name"`                   // Rule name for identification
	PathPatterns []string         `json:"path_patterns" yaml:"path_patterns"` // Array of regex patterns for path matching
	Levels       []RateLimitLevel `json:"levels" yaml:"levels"`               // Multiple rate limit levels
}

// RateLimitLevel defines a single rate limit level
type RateLimitLevel struct {
	Name          string   `json:"name" yaml:"name"`                     // Level name (e.g., "global", "ip", "device")
	MaxRequests   int      `json:"max_requests" yaml:"max_requests"`     // Maximum requests per minute
	KeyAttributes []string `json:"key_attributes" yaml:"key_attributes"` // Attributes to use as rate limit key
	BurstSize     int      `json:"burst_size" yaml:"burst_size"`         // Burst size for rate limiting
}

// MultiLevelConfig contains all multi-level rate limiting configuration
type MultiLevelConfig struct {
	Rules []MultiLevelRule `json:"rules" yaml:"rules"`
}

// MultiLevelLimiter implements multi-level rate limiting
type MultiLevelLimiter struct {
	config   *MultiLevelConfig
	rules    []*CompiledMultiLevelRule
	redis    *redis.Client
	limiter  *redis_rate.Limiter
	useRedis bool
}

// CompiledMultiLevelRule represents a compiled multi-level rule
type CompiledMultiLevelRule struct {
	Name         string
	PathPatterns []string
	Levels       []*CompiledLevel
}

// CompiledLevel represents a compiled rate limit level
type CompiledLevel struct {
	Name          string
	MaxRequests   int
	KeyAttributes []string
	BurstSize     int
	rate          redis_rate.Limit
	limiters      map[string]*TokenBucket
	mu            sync.RWMutex
}

// NewMultiLevelLimiter creates a new multi-level rate limiter (in-memory)
func NewMultiLevelLimiter(config *MultiLevelConfig) (*MultiLevelLimiter, error) {
	return newMultiLevelLimiter(config, nil, false)
}

// NewMultiLevelLimiterWithRedis creates a new Redis-based multi-level rate limiter
func NewMultiLevelLimiterWithRedis(config *MultiLevelConfig, client *redis.Client) (*MultiLevelLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	return newMultiLevelLimiter(config, client, true)
}

// newMultiLevelLimiter creates a multi-level rate limiter with optional Redis support
func newMultiLevelLimiter(config *MultiLevelConfig, client *redis.Client, useRedis bool) (*MultiLevelLimiter, error) {
	limiter := &MultiLevelLimiter{
		config:   config,
		rules:    make([]*CompiledMultiLevelRule, 0, len(config.Rules)),
		redis:    client,
		useRedis: useRedis,
	}

	if useRedis {
		limiter.limiter = redis_rate.NewLimiter(client)
	}

	for _, rule := range config.Rules {
		compiledRule, err := limiter.compileMultiLevelRule(&rule)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %s: %w", rule.Name, err)
		}
		limiter.rules = append(limiter.rules, compiledRule)
	}

	return limiter, nil
}

// compileMultiLevelRule compiles a multi-level rule
func (mll *MultiLevelLimiter) compileMultiLevelRule(rule *MultiLevelRule) (*CompiledMultiLevelRule, error) {
	compiledRule := &CompiledMultiLevelRule{
		Name:         rule.Name,
		PathPatterns: rule.PathPatterns,
		Levels:       make([]*CompiledLevel, 0, len(rule.Levels)),
	}

	for _, level := range rule.Levels {
		compiledLevel := &CompiledLevel{
			Name:          level.Name,
			MaxRequests:   level.MaxRequests,
			KeyAttributes: level.KeyAttributes,
			BurstSize:     level.BurstSize,
			rate:          redis_rate.PerMinute(level.MaxRequests),
			limiters:      make(map[string]*TokenBucket),
		}
		compiledRule.Levels = append(compiledRule.Levels, compiledLevel)
	}

	return compiledRule, nil
}

// Limit checks if the request should be rate limited based on multi-level rules
func (mll *MultiLevelLimiter) Limit(ctx context.Context) error {
	// Get request attributes from context
	attrs, ok := attributes.GetFromContext(ctx)
	if !ok {
		// If no attributes found, allow the request
		return nil
	}

	path := attrs.Path

	// Find matching rule
	var matchingRule *CompiledMultiLevelRule
	for _, rule := range mll.rules {
		for _, pattern := range rule.PathPatterns {
			if strings.Contains(path, pattern) || pattern == "*" {
				matchingRule = rule
				break
			}
		}
		if matchingRule != nil {
			break
		}
	}

	if matchingRule == nil {
		// No matching rule, allow the request
		return nil
	}

	// Check all levels for this rule
	for _, level := range matchingRule.Levels {
		if err := mll.checkLevel(ctx, matchingRule.Name, level, attrs); err != nil {
			return err
		}
	}

	return nil
}

// checkLevel checks a single rate limit level
func (mll *MultiLevelLimiter) checkLevel(ctx context.Context, ruleName string, level *CompiledLevel, attrs *attributes.RequestAttributes) error {
	// Generate rate limit key for this level
	key := mll.generateLevelKey(ruleName, level.Name, attrs, level.KeyAttributes)
	if key == "" {
		// If no key can be generated, allow the request
		return nil
	}

	if mll.useRedis {
		return mll.checkLevelWithRedis(ctx, level, key)
	}
	return mll.checkLevelWithMemory(level, key)
}

// checkLevelWithRedis performs rate limiting using Redis
func (mll *MultiLevelLimiter) checkLevelWithRedis(ctx context.Context, level *CompiledLevel, key string) error {
	// Add level prefix to Redis key
	redisKey := fmt.Sprintf("ratelimit:multilevel:%s", key)

	// Check rate limit using Redis
	res, err := mll.limiter.Allow(ctx, redisKey, level.rate)
	if err != nil {
		// If Redis is unavailable, allow the request (fail open)
		return nil
	}

	if res.Allowed == 0 {
		return fmt.Errorf("rate limit exceeded for level '%s' with key: %s", level.Name, key)
	}

	return nil
}

// checkLevelWithMemory performs rate limiting using in-memory token bucket
func (mll *MultiLevelLimiter) checkLevelWithMemory(level *CompiledLevel, key string) error {
	// Get or create token bucket for this key
	level.mu.Lock()
	limiter, exists := level.limiters[key]
	if !exists {
		// Create new token bucket with rate per second
		ratePerSecond := float64(level.MaxRequests) / 60.0
		limiter = NewTokenBucket(level.BurstSize, ratePerSecond)
		level.limiters[key] = limiter
	}
	level.mu.Unlock()

	// Check if request is allowed
	if !limiter.Take() {
		return fmt.Errorf("rate limit exceeded for level '%s' with key: %s", level.Name, key)
	}

	return nil
}

// generateLevelKey generates a rate limit key for a specific level
func (mll *MultiLevelLimiter) generateLevelKey(ruleName, levelName string, attrs *attributes.RequestAttributes, keyAttributes []string) string {
	var keyParts []string
	for _, attr := range keyAttributes {
		var value string
		switch strings.ToLower(attr) {
		case "ip":
			value = attrs.IP
		case "user_id":
			value = attrs.UserID
		case "staff_id":
			value = attrs.StaffID
		case "device_id":
			value = attrs.DeviceID
		case "global":
			value = "global" // Global level uses same key for all requests
		default:
			// Try to get from headers
			value = attrs.Get(attr)
		}

		if value != "" {
			keyParts = append(keyParts, value)
		}
	}

	if len(keyParts) == 0 {
		return fmt.Sprintf("%s:%s:default", ruleName, levelName)
	}

	return fmt.Sprintf("%s:%s:%s", ruleName, levelName, strings.Join(keyParts, ":"))
}

// ResetLevelRateLimit resets the rate limit for a specific level and key (Redis only)
func (mll *MultiLevelLimiter) ResetLevelRateLimit(ctx context.Context, ruleName, levelName, key string) error {
	if !mll.useRedis {
		return fmt.Errorf("reset rate limit is only available for Redis-based limiters")
	}

	// Delete the key from Redis to reset the rate limit
	redisKey := fmt.Sprintf("ratelimit:multilevel:%s:%s:%s", ruleName, levelName, key)
	return mll.redis.Del(ctx, redisKey).Err()
}

// Close closes the Redis client if using Redis
func (mll *MultiLevelLimiter) Close() error {
	if mll.useRedis && mll.redis != nil {
		return mll.redis.Close()
	}
	return nil
}

// IsRedisEnabled returns true if Redis is being used
func (mll *MultiLevelLimiter) IsRedisEnabled() bool {
	return mll.useRedis
}
