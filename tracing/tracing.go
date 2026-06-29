package tracing

import (
	"os"
	"strings"
)

type TracingConfig struct {
	ServiceName        string `mapstructure:"service_name" yaml:"service_name"`
	EnableTracing      bool   `mapstructure:"enable_tracing" yaml:"enable_tracing"`
	EnableDBTracing    bool   `mapstructure:"enable_db_tracing" yaml:"enable_db_tracing"`
	EnableRedisTracing bool   `mapstructure:"enable_redis_tracing" yaml:"enable_redis_tracing"`
	EnableKafkaTracing bool   `mapstructure:"enable_kafka_tracing" yaml:"enable_kafka_tracing"`
	EnableHTTPTracing  bool   `mapstructure:"enable_http_tracing" yaml:"enable_http_tracing"`
	EnableGRPCTracing  bool   `mapstructure:"enable_grpc_tracing" yaml:"enable_grpc_tracing"`
	GrpcEndpoint       string `mapstructure:"grpc_endpoint" yaml:"grpc_endpoint"`
}

func (t *TracingConfig) IsEnableTracing() bool {
	return t.EnableTracing
}

func (t *TracingConfig) IsEnableDBTracing() bool {
	return t.EnableTracing && t.EnableDBTracing
}

func (t *TracingConfig) IsEnableRedisTracing() bool {
	return t.EnableTracing && t.EnableRedisTracing
}

func (t *TracingConfig) IsEnableKafkaTracing() bool {
	return t.EnableTracing && t.EnableKafkaTracing
}

func (t *TracingConfig) IsEnableHTTPTracing() bool {
	return t.EnableTracing && t.EnableHTTPTracing
}

func (t *TracingConfig) IsEnableGRPCTracing() bool {
	return t.EnableTracing && t.EnableGRPCTracing
}

func getBoolFromEnv(env string, defaultValue bool) bool {
	if os.Getenv(env) == "" {
		return defaultValue
	}
	return strings.ToUpper(os.Getenv(env)) == "TRUE"
}

func loadTracingConfigFromEnv() *TracingConfig {
	return &TracingConfig{
		ServiceName:        os.Getenv("SERVICE_NAME"),
		EnableTracing:      getBoolFromEnv("ENABLE_TRACING", false),
		EnableDBTracing:    getBoolFromEnv("ENABLE_DB_TRACING", false),
		EnableRedisTracing: getBoolFromEnv("ENABLE_REDIS_TRACING", false),
		EnableKafkaTracing: getBoolFromEnv("ENABLE_KAFKA_TRACING", false),
		EnableHTTPTracing:  getBoolFromEnv("ENABLE_HTTP_TRACING", false),
		EnableGRPCTracing:  getBoolFromEnv("ENABLE_GRPC_TRACING", false),
	}
}

var globalTracingConfig *TracingConfig

func init() {
	globalTracingConfig = loadTracingConfigFromEnv()
}

func SetGlobalTracingConfig(cfg *TracingConfig) {
	globalTracingConfig = cfg
}

func GetGlobalTracingConfig() *TracingConfig {
	return globalTracingConfig
}
