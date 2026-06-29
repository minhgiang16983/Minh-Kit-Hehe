package infra

import (
	"context"
	"fmt"

	"github.com/minhgiang16983/service-kit/config"
	"github.com/minhgiang16983/Minh-Kit-Hehe/db"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kafka"
	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/redis"
	"gorm.io/gorm"
)

// InfraInterface defines methods to access infrastructure
type InfraInterface interface {
	Components() []lifecycle.Component
	GetMariaDb() *gorm.DB
	GetRedisStore() *redis.RedisStore
	GetKafkaClient() *kafka.KafkaClient
	Close() error
	GetConfig() *config.Config
}

// Infra manages connections to MariaDB, Redis and Kafka
type Infra struct {
	mariaDb     *db.Client
	redisStore  *redis.RedisStore
	kafkaClient *kafka.KafkaClient
	l           logger.LoggerInterface
	config      *config.Config
}

// New initializes Infra with connections to MariaDB, Redis and Kafka
func New(config *config.Config, l logger.LoggerInterface) (InfraInterface, error) {
	dbConfig := config.MariaDB
	if dbConfig.LogLevel == "" {
		dbConfig.LogLevel = config.LogLevel
	}
	mariaDb, err := db.NewClient(&dbConfig)
	if err != nil {
		l.Error("Failed to initialize MariaDB", l.String("error", err.Error()))
		return nil, err
	}
	l.Info("Connected to MariaDB", l.String("host", config.MariaDB.Host), l.Int("port", config.MariaDB.Port))

	redisStore, err := redis.New(&config.Redis)
	if err != nil {
		l.Error("Failed to initialize RedisStore", l.String("error", err.Error()))
		return nil, err
	}
	l.Info("Connected to RedisStore", l.String("host", config.Redis.Host), l.Int("port", config.Redis.Port))

	kafkaInterface, err := kafka.New(&config.Kafka, l)
	if err != nil {
		l.Error("Failed to initialize KafkaClient", l.String("error", err.Error()))
		return nil, err
	}
	kafkaClient, ok := kafkaInterface.(*kafka.KafkaClient)
	if !ok {
		return nil, fmt.Errorf("failed to assert kafka client type")
	}
	l.Info("Connected to KafkaClient", l.String("brokers", config.Kafka.Brokers))

	return &Infra{
		mariaDb:     mariaDb,
		redisStore:  redisStore,
		kafkaClient: kafkaClient,
		l:           l,
		config:      config,
	}, nil
}

func (i *Infra) Components() []lifecycle.Component {
	return []lifecycle.Component{i.mariaDb, i.redisStore, i.kafkaClient}
}

func (i *Infra) GetMariaDb() *gorm.DB {
	return i.mariaDb.DB
}

func (i *Infra) GetRedisStore() *redis.RedisStore {
	return i.redisStore
}

func (i *Infra) GetKafkaClient() *kafka.KafkaClient {
	return i.kafkaClient
}

func (i *Infra) Close() error {
	group := lifecycle.NewGroup("infra", i.Components()...)
	return group.Stop(context.Background())
}

func (i *Infra) GetConfig() *config.Config {
	return i.config
}
