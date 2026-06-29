package infra

import (
	"context"
	"fmt"

	"github.com/minhgiang16983/service-kit/config"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kafka"
	"github.com/minhgiang16983/Minh-Kit-Hehe/db"
	"github.com/minhgiang16983/Minh-Kit-Hehe/redis"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"gorm.io/gorm"
)

// InfraInterface defines methods to access infrastructure
type InfraInterface interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetMariaDb() *gorm.DB
	GetRedisStore() *redis.RedisStore
	GetKafkaClient() *kafka.KafkaClient
	Close() error
	GetConfig() *config.Config
}

// Infra manages connections to MariaDB, Redis and Kafka
type Infra struct {
	mariaDb     *gorm.DB
	redisStore  *redis.RedisStore
	kafkaClient *kafka.KafkaClient
	l           logger.LoggerInterface
	config      *config.Config
}

// New initializes Infra with connections to MariaDB, Redis and Kafka
func New(config *config.Config, l logger.LoggerInterface) (InfraInterface, error) {
	// Initialize MariaDB
	// Set LogLevel in DBConfig if not already set
	dbConfig := config.MariaDB
	if dbConfig.LogLevel == "" {
		dbConfig.LogLevel = config.LogLevel
	}
	mariaDb, err := db.New(&dbConfig)
	if err != nil {
		l.Error("Failed to initialize MariaDB", l.String("error", err.Error()))
		return nil, err
	}
	l.Info("Connected to MariaDB", l.String("host", config.MariaDB.Host), l.Int("port", config.MariaDB.Port))

	// Initialize Redis
	redisStore, err := redis.New(&config.Redis)
	if err != nil {
		l.Error("Failed to initialize RedisStore", l.String("error", err.Error()))
		return nil, err
	}
	l.Info("Connected to RedisStore", l.String("host", config.Redis.Host), l.Int("port", config.Redis.Port))

	// Initialize Kafka
	kafkaInterface, err := kafka.New(&config.Kafka, l)
	if err != nil {
		l.Error("Failed to initialize KafkaClient", l.String("error", err.Error()))
		return nil, err
	}
	// Type assert to *KafkaClient
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

// GetMariaDb returns MariaDB connection
func (i *Infra) GetMariaDb() *gorm.DB {
	return i.mariaDb
}

// GetRedisStore returns RedisStore
func (i *Infra) GetRedisStore() *redis.RedisStore {
	return i.redisStore
}

// GetKafkaClient returns KafkaClient
func (i *Infra) GetKafkaClient() *kafka.KafkaClient {
	return i.kafkaClient
}

// Name returns the lifecycle component name.
func (i *Infra) Name() string { return "infra" }

// Start verifies infrastructure is ready.
func (i *Infra) Start(_ context.Context) error {
	i.l.Info("infrastructure ready")
	return nil
}

// Stop closes all infrastructure connections.
func (i *Infra) Stop(ctx context.Context) error {
	return i.Close()
}

// Close closes all connections
func (i *Infra) Close() error {
	var errs []error

	// Close Kafka
	if i.kafkaClient != nil {
		i.kafkaClient.Close()
	}

	// Close Redis
	if err := i.redisStore.Client.Close(); err != nil {
		i.l.Error("Failed to close RedisStore", i.l.String("error", err.Error()))
		errs = append(errs, err)
	}

	// Close MariaDB
	sqlDB, err := i.mariaDb.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			i.l.Error("Failed to close MariaDB", i.l.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close resources: %v", errs)
	}
	return nil
}

func (i *Infra) GetConfig() *config.Config {
	return i.config
}