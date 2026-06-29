# Rate Limit Middleware

A flexible rate limiting middleware for both gRPC and HTTP services that supports regex path matching and attribute-based rate limiting with both in-memory and distributed Redis-based implementations.

## Features

- **Regex Path Matching**: Define rate limit rules using regex patterns
- **Attribute-Based Keys**: Use request attributes (IP, User ID, Staff ID, Device ID, custom headers) as rate limit keys
- **Rule Groups**: Group multiple paths together to share the same rate limit quota
- **Multiple Backends**: Choose between in-memory token bucket or distributed Redis-based rate limiting
- **Distributed Rate Limiting**: Use Redis for rate limiting across multiple server instances
- **Token Bucket Algorithm**: Implements a token bucket rate limiter for smooth rate limiting
- **Burst Support**: Configure burst sizes for handling traffic spikes
- **gRPC & HTTP Support**: Works with both gRPC interceptors and HTTP middleware
- **Flexible Configuration**: Multiple rules with different configurations
- **Fail-Open Design**: Gracefully handles Redis unavailability

## Quick Start

### Simple HTTP Rate Limiting

```go
package main

import (
    "net/http"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    // Simple rate limiting: 100 requests per minute per IP
    handler := ratelimit.SimpleHTTPMiddleware(100)
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    
    // Apply rate limiting middleware
    rateLimitedHandler := handler(mux)
    
    http.ListenAndServe(":8080", rateLimitedHandler)
}
```

### Advanced Configuration

```go
package main

import (
    "net/http"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    config := &ratelimit.RateLimitConfig{
        Rules: []ratelimit.RateLimitRule{
            {
                PathPattern:   "/api/v1/users/.*",           // Match user endpoints
                MaxRequests:   60,                           // 60 requests per minute
                KeyAttributes: []string{"user_id", "ip"},    // Use user_id + IP as key
                BurstSize:     5,                            // Allow burst of 5 requests
            },
            {
                PathPattern:   "/api/v1/admin/.*",           // Match admin endpoints
                MaxRequests:   30,                           // 30 requests per minute
                KeyAttributes: []string{"staff_id", "ip"},   // Use staff_id + IP as key
                BurstSize:     3,
            },
            {
                PathPattern:   "/api/v1/public/.*",          // Match public endpoints
                MaxRequests:   200,                          // 200 requests per minute
                KeyAttributes: []string{"ip"},               // Use only IP as key
                BurstSize:     20,
            },
        },
    }
    
    limiter, err := ratelimit.NewAttributeBasedLimiter(config)
    if err != nil {
        panic(err)
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/users/profile", userHandler)
    mux.HandleFunc("/api/v1/admin/dashboard", adminHandler)
    mux.HandleFunc("/api/v1/public/status", publicHandler)
    
    // Apply rate limiting middleware
    handler := ratelimit.HTTPMiddleware(limiter)(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

### gRPC Rate Limiting

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    config := &ratelimit.RateLimitConfig{
        Rules: []ratelimit.RateLimitRule{
            {
                PathPattern:   "/base_service.UserService/.*",
                MaxRequests:   100,
                KeyAttributes: []string{"user_id", "ip"},
                BurstSize:     10,
            },
        },
    }
    
    limiter, err := ratelimit.NewAttributeBasedLimiter(config)
    if err != nil {
        panic(err)
    }
    
    // Create gRPC server with rate limiting interceptor
    server := grpc.NewServer(
        grpc.UnaryInterceptor(ratelimit.UnaryServerInterceptor(limiter)),
    )
    
    // Register your services...
    // server.Serve(lis)
}
```

### Distributed Redis Rate Limiting

```go
package main

import (
    "net/http"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    // Create Redis client
    redisClient := ratelimit.NewRedisClient("localhost:6379", "", 0)
    defer redisClient.Close()

    config := &ratelimit.RedisRateLimitConfig{
        Rules: []ratelimit.RedisRateLimitRule{
            {
                PathPattern:   "/api/.*",
                MaxRequests:   60,
                KeyAttributes: []string{"user_id", "ip"},
                BurstSize:     5,
            },
        },
    }
    
    limiter, err := ratelimit.NewRedisAttributeBasedLimiter(config, redisClient)
    if err != nil {
        panic(err)
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })
    
    // Apply distributed rate limiting middleware
    handler := ratelimit.HTTPMiddleware(limiter)(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

### Rule Groups (Multiple Paths Sharing Same Rate Limit)

#### In-Memory Rule Groups

```go
package main

import (
    "net/http"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    // Group-based rate limiting - multiple paths share the same quota
    config := &ratelimit.RuleGroupConfig{
        Groups: []ratelimit.RuleGroup{
            {
                Name:         "token_operations",
                MaxRequests:  30, // 30 requests per minute for all token operations
                KeyAttributes: []string{"user_id", "ip"},
                BurstSize:    5,
                Paths: []string{
                    "/api/v1/token",                    // HTTP token endpoint
                    "/base_service.TokenService/CreateToken", // gRPC token creation
                    "/base_service.TokenService/RefreshToken", // gRPC token refresh
                    "/base_service.TokenService/RevokeToken",  // gRPC token revocation
                },
            },
        },
    }
    
    limiter, err := ratelimit.NewGroupBasedLimiter(config)
    if err != nil {
        panic(err)
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/token", tokenHandler)
    
    // Apply group-based rate limiting middleware
    handler := ratelimit.HTTPMiddleware(limiter)(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

#### Redis-Based Rule Groups (Distributed)

```go
package main

import (
    "net/http"
    "github.com/minhgiang16983/Minh-Kit-Hehe/ratelimit"
)

func main() {
    // Create Redis client
    redisClient := ratelimit.NewRedisClient("localhost:6379", "", 0)
    defer redisClient.Close()

    // Group-based rate limiting with Redis - multiple paths share the same quota
    config := &ratelimit.RuleGroupConfig{
        Groups: []ratelimit.RuleGroup{
            {
                Name:         "token_operations",
                MaxRequests:  30, // 30 requests per minute for all token operations
                KeyAttributes: []string{"user_id", "ip"},
                BurstSize:    5,
                Paths: []string{
                    "/api/v1/token",                    // HTTP token endpoint
                    "/base_service.TokenService/CreateToken", // gRPC token creation
                    "/base_service.TokenService/RefreshToken", // gRPC token refresh
                    "/base_service.TokenService/RevokeToken",  // gRPC token revocation
                },
            },
        },
    }
    
    limiter, err := ratelimit.NewGroupBasedLimiterWithRedis(config, redisClient)
    if err != nil {
        panic(err)
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/token", tokenHandler)
    
    // Apply distributed group-based rate limiting middleware
    handler := ratelimit.HTTPMiddleware(limiter)(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

## Configuration

### In-Memory Rate Limiting

#### RateLimitRule

```go
type RateLimitRule struct {
    PathPattern    string   `json:"path_pattern"`    // Regex pattern for path matching
    MaxRequests    int      `json:"max_requests"`    // Maximum requests per minute
    KeyAttributes  []string `json:"key_attributes"`  // Attributes to use as rate limit key
    BurstSize      int      `json:"burst_size"`      // Burst size for rate limiting
}
```

### Distributed Redis Rate Limiting

#### RedisRateLimitRule

```go
type RedisRateLimitRule struct {
    PathPattern   string   `json:"path_pattern"`   // Regex pattern for path matching
    MaxRequests   int      `json:"max_requests"`   // Maximum requests per minute
    KeyAttributes []string `json:"key_attributes"` // Attributes to use as rate limit key
    BurstSize     int      `json:"burst_size"`     // Burst size for rate limiting
}
```

### Rule Groups

#### RuleGroup (In-Memory)

```go
type RuleGroup struct {
    Name         string   `json:"name"`         // Group name for identification
    MaxRequests  int      `json:"max_requests"` // Maximum requests per minute for the group
    KeyAttributes []string `json:"key_attributes"` // Attributes to use as rate limit key
    BurstSize    int      `json:"burst_size"`   // Burst size for the group
    Paths        []string `json:"paths"`        // List of path patterns (regex) for this group
}
```

#### RedisRuleGroup (Distributed)

```go
type RedisRuleGroup struct {
    Name         string   `json:"name"`         // Group name for identification
    MaxRequests  int      `json:"max_requests"` // Maximum requests per minute for the group
    KeyAttributes []string `json:"key_attributes"` // Attributes to use as rate limit key
    BurstSize    int      `json:"burst_size"`   // Burst size for the group
    Paths        []string `json:"paths"`        // List of path patterns (regex) for this group
}
```

### Supported Key Attributes

- `ip`: Client IP address
- `user_id`: User ID from `X-User-ID` header
- `staff_id`: Staff ID from `X-Staff-ID` header
- `device_id`: Device ID from `X-Device-ID` header
- Any custom header name (e.g., `x-api-key`)

### Path Pattern Examples

- `.*`: Match all paths
- `/api/.*`: Match all API paths
- `/api/v1/users/.*`: Match user-related endpoints
- `/service\.(User|Admin)Service/.*`: Match specific gRPC services
- `^/api/v1/(users|admin)/.*$`: Match specific API endpoints

## How It Works

### In-Memory Rate Limiting
1. **Attribute Extraction**: The middleware extracts request attributes (IP, headers, etc.) using the attributes package
2. **Path Matching**: Uses regex to match the request path against configured rules
3. **Key Generation**: Combines specified attributes to create a unique rate limit key
4. **Token Bucket**: Uses a token bucket algorithm to enforce rate limits
5. **Rate Limiting**: Allows or rejects requests based on available tokens

### Distributed Redis Rate Limiting
1. **Attribute Extraction**: Same as in-memory version
2. **Path Matching**: Same as in-memory version
3. **Key Generation**: Creates Redis keys with `ratelimit:` prefix
4. **Redis GCRA**: Uses Redis with GCRA (Generic Cell Rate Algorithm) for distributed rate limiting
5. **Fail-Open**: Gracefully handles Redis unavailability by allowing requests

### Rule Groups
1. **Group Definition**: Define groups with multiple paths that share the same rate limit
2. **Path Matching**: Match request paths against all paths in all groups
3. **Shared Quota**: All paths in a group share the same rate limit quota
4. **Key Generation**: Generate rate limit keys based on group configuration
5. **Rate Limiting**: Apply rate limiting at the group level

## Rate Limiting Algorithms

### Token Bucket Algorithm (In-Memory)
The in-memory rate limiter uses a token bucket algorithm where:
- Tokens are refilled at a rate of `MaxRequests / 60` per second
- The bucket can hold up to `BurstSize` tokens
- Each request consumes one token
- If no tokens are available, the request is rejected

### GCRA Algorithm (Redis)
The Redis-based rate limiter uses GCRA (Generic Cell Rate Algorithm) where:
- Rate limits are enforced using Redis with atomic operations
- Provides distributed rate limiting across multiple server instances
- Automatically handles rate limit expiration
- More accurate for distributed scenarios

## Error Handling

When rate limits are exceeded:
- **HTTP**: Returns `429 Too Many Requests` status code
- **gRPC**: Returns `ResourceExhausted` error code

## Best Practices

1. **Use Specific Paths**: Instead of `.*`, use specific path patterns for better control
2. **Combine Attributes**: Use multiple attributes (e.g., `user_id + ip`) for more granular control
3. **Set Appropriate Burst Sizes**: Burst sizes should be smaller than max requests for smooth limiting
4. **Monitor Usage**: Track rate limit hits to adjust configurations
5. **Test Patterns**: Verify regex patterns work as expected
6. **Choose the Right Backend**: Use in-memory for single instances, Redis for distributed deployments
7. **Redis Configuration**: Ensure Redis is properly configured for your rate limiting needs
8. **Fail-Open Strategy**: The Redis implementation fails open when Redis is unavailable

## Examples

See the following files for comprehensive usage examples:
- `example.go` - In-memory rate limiting examples
- `redis_example.go` - Redis-based distributed rate limiting examples
- `group_example.go` - Rule groups for in-memory rate limiting
- `redis_group_example.go` - Rule groups for Redis-based distributed rate limiting

### Example Features:
- Simple rate limiting
- Advanced multi-rule configurations
- Custom header-based rate limiting
- gRPC server setup
- HTTP server setup
- Distributed rate limiting across multiple instances
- Redis client configuration
- Rate limit management and reset
- **Rule Groups**: Multiple paths sharing the same rate limit quota
- **Mixed Protocols**: HTTP and gRPC endpoints in the same group
- **Group Management**: List and inspect rule groups 