package config

import (
	"bytes"
	"strings"

	"github.com/minhgiang16983/Minh-Kit-Hehe/db"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kafka"
	"github.com/minhgiang16983/Minh-Kit-Hehe/redis"

	"github.com/spf13/viper"
)

var defaultConfig = `
environment: D
grpc_port: 10000
http_port: 9000
log_level: debug
mariadb:
  username: service-kit
  password: "service-kit-password"
  host: 127.0.0.1   # DB host
  port: 3306
  database_name: service-kit
redis:
  host: 127.0.0.1 # Redis host
  port: 6379
  password: admin@123
  db: 1
kafka:
  brokers: "localhost:9092"
`

type Config struct {
	Environment string      `mapstructure:"environment" yaml:"environment"`
	GrpcPort    int         `mapstructure:"grpc_port" yaml:"grpc_port"`
	HttpPort    int         `mapstructure:"http_port" yaml:"http_port"`
	LogLevel    string      `mapstructure:"log_level" yaml:"log_level"`
	MariaDB     db.DBConfig     `mapstructure:"mariadb" yaml:"mariadb"`
	Redis       redis.RedisConfig       `mapstructure:"redis" yaml:"redis"`
	Kafka       kafka.KafkaConfig `mapstructure:"kafka" yaml:"kafka"`
}

func Load() (*Config, error) {
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(bytes.NewBuffer([]byte(defaultConfig)))
	if err != nil {
		return nil, err
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))

	cfg := Config{}
	if err = viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readConfig() {
}
