package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/IBM/sarama"
)

var _ lifecycle.Component = (*ConsumerComponent)(nil)

// ConsumerComponent runs a Kafka consumer group loop with lifecycle control.
type ConsumerComponent struct {
	client       *KafkaClient
	consumerName string
	topics       []string
	handler      sarama.ConsumerGroupHandler

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewConsumerComponent(
	client *KafkaClient,
	consumerName string,
	topics []string,
	handler sarama.ConsumerGroupHandler,
) *ConsumerComponent {
	return &ConsumerComponent{
		client:       client,
		consumerName: consumerName,
		topics:       topics,
		handler:      handler,
	}
}

func NewMessageConsumerComponent(
	client *KafkaClient,
	consumerName string,
	topics []string,
	handler ConsumeMessageHandler,
) *ConsumerComponent {
	return NewConsumerComponent(client, consumerName, topics, &defaultConsumerGroupHandler{
		handler: handler,
		logger:  client.Logger,
	})
}

func (c *ConsumerComponent) Name() string {
	return fmt.Sprintf("kafka-consumer:%s", c.consumerName)
}

func (c *ConsumerComponent) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.client.consumeLoop(runCtx, c.consumerName, c.topics, c.handler); err != nil && runCtx.Err() == nil {
			c.client.Logger.Error("kafka consumer stopped", c.client.Logger.String("consumer", c.consumerName), c.client.Logger.ErrorField(err))
		}
	}()

	return nil
}

func (c *ConsumerComponent) Stop(ctx context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
