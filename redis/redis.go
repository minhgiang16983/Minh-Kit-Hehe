package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisotel/v9"
	redisprometheus "github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host                  string `mapstructure:"host" yaml:"host"`
	Port                  int    `mapstructure:"port" yaml:"port"`
	Password              string `mapstructure:"password" yaml:"password"`
	Username              string `mapstructure:"username" yaml:"username"`
	DB                    int    `mapstructure:"db" yaml:"db"`
	EnableTracing         bool   `mapstructure:"enable_tracing" yaml:"enable_tracing"`
	MaxIdleConns          int    `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	MaxActiveConns        int    `mapstructure:"max_active_conns" yaml:"max_active_conns"`
	TLSEnabled            bool   `mapstructure:"tls_enabled" yaml:"tls_enabled"`
	TLSCAFile             string `mapstructure:"tls_ca_file" yaml:"tls_ca_file"`
	TLSCertFile           string `mapstructure:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile            string `mapstructure:"tls_key_file" yaml:"tls_key_file"`
	TLSServerName         string `mapstructure:"tls_server_name" yaml:"tls_server_name"`
	TLSInsecureSkipVerify bool   `mapstructure:"tls_insecure_skip_verify" yaml:"tls_insecure_skip_verify"`
}

type RedisStore struct {
	*redis.Client
}

func New(config *RedisConfig) (*RedisStore, error) {
	logger := logger.GetLogger(context.Background())
	redisConfig := config

	options := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
		Password:        redisConfig.Password,
		Username:        redisConfig.Username,
		DB:              redisConfig.DB,
		DisableIdentity: true, // for redis <v7.2, disable identity to avoid "redis: both LibName and LibVer cannot be set at the same time" error
		MaxIdleConns:    redisConfig.MaxIdleConns,
		MaxActiveConns:  redisConfig.MaxActiveConns,
	}
	logger.Info("Connecting to Redis", logger.String("host", redisConfig.Host), logger.Int("port", redisConfig.Port), logger.String("tls_enabled", strconv.FormatBool(redisConfig.TLSEnabled)), logger.String("tls_insecure_skip_verify", strconv.FormatBool(redisConfig.TLSInsecureSkipVerify)))

	if redisConfig.TLSEnabled {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: redisConfig.TLSInsecureSkipVerify,
		}
		if redisConfig.TLSCAFile != "" {
			caCert, err := os.ReadFile(redisConfig.TLSCAFile)
			if err != nil {
				return nil, err
			}
			options.TLSConfig.RootCAs = x509.NewCertPool()
			options.TLSConfig.RootCAs.AppendCertsFromPEM(caCert)
		}
		if redisConfig.TLSCertFile != "" && redisConfig.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(redisConfig.TLSCertFile, redisConfig.TLSKeyFile)
			if err != nil {
				return nil, err
			}
			options.TLSConfig.Certificates = []tls.Certificate{cert}
		}
		if redisConfig.TLSServerName != "" {
			options.TLSConfig.ServerName = redisConfig.TLSServerName
		}
	}
	rdb := redis.NewClient(options)

	collector := redisprometheus.NewCollector("redis", "client", rdb)
	prometheus.MustRegister(collector)
	if tracing.GetGlobalTracingConfig().IsEnableRedisTracing() {
		// Enable tracing instrumentation.
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			return nil, err
		}
	}

	// Ping pong to make sure redis connected
	i := 0
	connected := false

	ctx := context.Background()
	for {
		pong, _ := rdb.Ping(ctx).Result()
		if pong == "PONG" {
			connected = true
			break
		}
		if i == 3 {
			break
		}
		i++
		time.Sleep(time.Second + 2)
	}

	if !connected {
		return nil, errors.New("redis is not connected")
	}

	return &RedisStore{rdb}, nil
}
