package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"github.com/IBM/sarama"
	"github.com/dnwe/otelsarama"
	"go.opentelemetry.io/otel"
)

type KafkaConfig struct {
	// Comma-separated or list in YAML (Viper/mapstructure will handle both if you like)
	Brokers string `mapstructure:"brokers" yaml:"brokers"`

	// Optional consumer group ID for consumers
	GroupID string `mapstructure:"group_id" yaml:"group_id"`

	// Optional auth (omit or set enabled:false to skip)
	Auth *KafkaAuth `mapstructure:"auth" yaml:"auth"`

	// Optional TLS (omit or set enabled:false to skip)
	TLS *KafkaTLS `mapstructure:"tls" yaml:"tls"`

	// Client tuning (optional)
	ClientID string `mapstructure:"client_id" yaml:"client_id"`
}

type KafkaAuth struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Mechanism string `mapstructure:"mechanism" yaml:"mechanism"` // "SCRAM-SHA-512" or "SCRAM-SHA-256"
	Username  string `mapstructure:"username" yaml:"username"`
	Password  string `mapstructure:"password" yaml:"password"`
	// If true and TLS is disabled, uses SASL_PLAINTEXT. If TLS enabled, uses SASL_SSL automatically.
	RequireSASL bool `mapstructure:"require_sasl" yaml:"require_sasl"`
}

type KafkaTLS struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	// If empty -> use system trust store (good when broker cert is public-CA signed)
	CAFile string `mapstructure:"ca_file" yaml:"ca_file"`
	// Optional: override SNI if needed
	ServerName string `mapstructure:"server_name" yaml:"server_name"`
	// Dev-only: skip cert validation (DON'T use in prod)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

type ConsumeMessageHandler func(ctx context.Context, msg *sarama.ConsumerMessage) error

// KafkaInterface defines standard methods for interacting with Kafka
type KafkaInterface interface {
	InitAsyncProducer(name string, config *sarama.Config) (sarama.AsyncProducer, error)
	InitProducer(name string, config *sarama.Config) (sarama.SyncProducer, error)
	InitConsumer(name, groupId string, config *sarama.Config) (sarama.ConsumerGroup, error)
	GetProducer(name string) (sarama.SyncProducer, bool)
	GetAsyncProducer(name string) (sarama.AsyncProducer, bool)
	GetConsumer(name string) (sarama.ConsumerGroup, bool)
	SendMessage(producerName, topic string, key, value []byte) (int32, int64, error)
	SendMessageAsync(ctx context.Context, producerName string, msg *sarama.ProducerMessage) error
	ConsumeLoop(consumerName string, topics []string, handler sarama.ConsumerGroupHandler) error
	ConsumeMessage(consumerName string, topics []string, handler ConsumeMessageHandler) error
	GetConfig() *sarama.Config
	HandleProducerResult(p sarama.AsyncProducer)
	Close()
}

// KafkaClient manages connection to Kafka
type KafkaClient struct {
	Brokers       []string
	AsyncProducer map[string]sarama.AsyncProducer
	Producer      map[string]sarama.SyncProducer
	Consumer      map[string]sarama.ConsumerGroup
	Logger        logger.LoggerInterface
	Config        *sarama.Config
	KafkaCfg      *KafkaConfig
}

// New creates and initializes a KafkaClient
func New(cfg *KafkaConfig, l logger.LoggerInterface) (KafkaInterface, error) {
	brokers := strings.Split(cfg.Brokers, ",")

	sarama.Logger = l

	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V2_1_0_0
	saramaCfg.MetricRegistry = NewPromRegistry()
	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRange()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll

	err := ApplyKafkaAuth(saramaCfg, cfg.Auth)
	if err != nil {
		return nil, err
	}

	err = ApplyKafkaTLS(saramaCfg, cfg.TLS)
	if err != nil {
		return nil, err
	}

	// Connection test
	for i := 0; i < 3; i++ {
		client, err := sarama.NewClient(brokers, saramaCfg)
		if err == nil {
			_ = client.Close()
			break
		}
		l.Warn("Failed to connect to Kafka", l.ErrorField(err))
		time.Sleep(2 * time.Second)
		if i == 2 {
			return nil, errors.New("kafka is not connected")
		}
	}

	return &KafkaClient{
		Brokers:       brokers,
		Logger:        l,
		AsyncProducer: make(map[string]sarama.AsyncProducer),
		Consumer:      make(map[string]sarama.ConsumerGroup),
		Producer:      make(map[string]sarama.SyncProducer),
		Config:        saramaCfg,
		KafkaCfg:      cfg,
	}, nil
}
func (k *KafkaClient) InitAsyncProducer(name string, config *sarama.Config) (sarama.AsyncProducer, error) {
	if config == nil {
		config = k.Config
	} else {
		config.Net.SASL = k.Config.Net.SASL
		config.Net.TLS = k.Config.Net.TLS
	}

	producer, err := sarama.NewAsyncProducer(k.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer %s: %w", name, err)
	}

	if tracing.GetGlobalTracingConfig().IsEnableKafkaTracing() {
		producer = otelsarama.WrapAsyncProducer(config, producer)
	}
	k.AsyncProducer[name] = producer

	return producer, nil
}

func (k *KafkaClient) InitProducer(name string, config *sarama.Config) (sarama.SyncProducer, error) {
	if config == nil {
		config = k.Config
	} else {
		config.Net.SASL = k.Config.Net.SASL
		config.Net.TLS = k.Config.Net.TLS
	}

	producer, err := sarama.NewSyncProducer(k.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer %s: %w", name, err)
	}

	if tracing.GetGlobalTracingConfig().IsEnableKafkaTracing() {
		producer = otelsarama.WrapSyncProducer(config, producer)
	}

	k.Producer[name] = producer
	return producer, nil
}

func (k *KafkaClient) InitConsumer(name, groupId string, config *sarama.Config) (sarama.ConsumerGroup, error) {
	if config == nil {
		config = k.Config
	} else {
		config.Net.SASL = k.Config.Net.SASL
		config.Net.TLS = k.Config.Net.TLS
	}
	consumer, err := sarama.NewConsumerGroup(k.Brokers, groupId, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer %s: %w", name, err)
	}
	k.Consumer[name] = consumer
	return consumer, nil
}

func (k *KafkaClient) GetAsyncProducer(name string) (sarama.AsyncProducer, bool) {
	p, ok := k.AsyncProducer[name]
	return p, ok
}

func (k *KafkaClient) GetProducer(name string) (sarama.SyncProducer, bool) {
	p, ok := k.Producer[name]
	return p, ok
}

func (k *KafkaClient) GetConsumer(name string) (sarama.ConsumerGroup, bool) {
	c, ok := k.Consumer[name]
	return c, ok
}

func (k *KafkaClient) SendMessage(producerName, topic string, key, value []byte) (int32, int64, error) {
	p, ok := k.GetProducer(producerName)
	if !ok {
		return 0, 0, fmt.Errorf("producer %s not found", producerName)
	}
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	return p.SendMessage(msg)
}

func (k *KafkaClient) ConsumeLoop(consumerName string, topics []string, handler sarama.ConsumerGroupHandler) error {
	if tracing.GetGlobalTracingConfig().IsEnableKafkaTracing() {
		handler = otelsarama.WrapConsumerGroupHandler(handler)
	}
	c, ok := k.GetConsumer(consumerName)
	if !ok {
		return fmt.Errorf("consumer %s not found", consumerName)
	}
	ctx := context.Background()
	go func() {
		for {
			if err := c.Consume(ctx, topics, handler); err != nil {
				k.Logger.Warn("Kafka consume error", k.Logger.String("error", err.Error()))
				time.Sleep(2 * time.Second)
			}
		}
	}()
	return nil
}

func (k *KafkaClient) HandleProducerResult(p sarama.AsyncProducer) {
	k.handleProducerResult(p, "")
}

func (k *KafkaClient) handleProducerResult(p sarama.AsyncProducer, name string) {
	go func() {
		for msg := range p.Successes() {
			k.Logger.Info("Kafka message sent", k.Logger.String("topic", msg.Topic), k.Logger.String("partition", fmt.Sprint(msg.Partition)))
		}
	}()
	go func() {
		for err := range p.Errors() {
			k.Logger.Error("Kafka message failed", k.Logger.String("topic", err.Msg.Topic), k.Logger.String("error", err.Err.Error()))
		}
	}()
}

func (k *KafkaClient) Close() {
	for name, p := range k.AsyncProducer {
		if err := p.Close(); err != nil {
			k.Logger.Info("Kafka producer close error", k.Logger.String("producer", name), k.Logger.String("error", err.Error()))
		}
	}
	for name, c := range k.Consumer {
		if err := c.Close(); err != nil {
			k.Logger.Info("Kafka consumer close error", k.Logger.String("consumer", name), k.Logger.String("error", err.Error()))
		}
	}
}

func (k *KafkaClient) SendMessageAsync(ctx context.Context, producerName string, msg *sarama.ProducerMessage) error {
	p, ok := k.GetAsyncProducer(producerName)
	if !ok {
		return fmt.Errorf("producer %s not found", producerName)
	}

	if tracing.GetGlobalTracingConfig().IsEnableKafkaTracing() {
		otel.GetTextMapPropagator().Inject(ctx, otelsarama.NewProducerMessageCarrier(msg))
	}

	p.Input() <- msg
	return nil
}

// ConsumeMessage starts consuming messages from the specified topics using the provided handler
func (k *KafkaClient) ConsumeMessage(consumerName string, topics []string, handler ConsumeMessageHandler) error {
	// Create a default ConsumerGroupHandler that wraps our custom handler
	consumerGroupHandler := &defaultConsumerGroupHandler{
		handler: handler,
		logger:  k.Logger,
	}

	// Use the existing ConsumeLoop method with our wrapped handler
	return k.ConsumeLoop(consumerName, topics, consumerGroupHandler)
}

// defaultConsumerGroupHandler implements sarama.ConsumerGroupHandler
type defaultConsumerGroupHandler struct {
	handler ConsumeMessageHandler
	logger  logger.LoggerInterface
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *defaultConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *defaultConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *defaultConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Create context from the session
			ctx := session.Context()

			if tracing.GetGlobalTracingConfig().IsEnableKafkaTracing() {
				ctx = otel.GetTextMapPropagator().Extract(ctx, otelsarama.NewConsumerMessageCarrier(message))
			}

			// Call the user-provided handler
			if err := h.handler(ctx, message); err != nil {
				h.logger.Error("Error processing message",
					h.logger.String("topic", message.Topic),
					h.logger.String("partition", fmt.Sprint(message.Partition)),
					h.logger.String("offset", fmt.Sprint(message.Offset)),
					h.logger.String("error", err.Error()))

				// Continue processing other messages even if one fails
				continue
			}

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (k *KafkaClient) GetConfig() *sarama.Config {
	return k.Config
}
