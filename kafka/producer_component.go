package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/IBM/sarama"
)

var _ lifecycle.Component = (*AsyncProducerComponent)(nil)

// AsyncProducerComponent monitors async producer success/error channels.
type AsyncProducerComponent struct {
	client       *KafkaClient
	producerName string

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewAsyncProducerComponent(client *KafkaClient, producerName string) *AsyncProducerComponent {
	return &AsyncProducerComponent{
		client:       client,
		producerName: producerName,
	}
}

func (c *AsyncProducerComponent) Name() string {
	return fmt.Sprintf("kafka-producer:%s", c.producerName)
}

func (c *AsyncProducerComponent) Start(ctx context.Context) error {
	producer, ok := c.client.GetAsyncProducer(c.producerName)
	if !ok {
		return fmt.Errorf("async producer %s not found", c.producerName)
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(2)
	go c.watchSuccesses(runCtx, producer)
	go c.watchErrors(runCtx, producer)

	return nil
}

func (c *AsyncProducerComponent) Stop(ctx context.Context) error {
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

func (c *AsyncProducerComponent) watchSuccesses(ctx context.Context, producer sarama.AsyncProducer) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-producer.Successes():
			if !ok {
				return
			}
			c.client.Logger.Info(
				"kafka message sent",
				c.client.Logger.String("producer", c.producerName),
				c.client.Logger.String("topic", msg.Topic),
				c.client.Logger.String("partition", fmt.Sprint(msg.Partition)),
			)
		}
	}
}

func (c *AsyncProducerComponent) watchErrors(ctx context.Context, producer sarama.AsyncProducer) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case errMsg, ok := <-producer.Errors():
			if !ok {
				return
			}
			c.client.Logger.Error(
				"kafka message failed",
				c.client.Logger.String("producer", c.producerName),
				c.client.Logger.String("topic", errMsg.Msg.Topic),
				c.client.Logger.String("error", errMsg.Err.Error()),
			)
		}
	}
}
