package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Consumer struct {
	consumer sarama.ConsumerGroup
	topic    string
	logger   *zap.Logger
	handler  func(ctx context.Context, event domain.StockReservedEvent) error
}

func NewConsumer(brokers []string, groupID string, topic string, logger *zap.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Enable = false

	consumer, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		topic:    topic,
		logger:   logger,
	}, nil
}

// ConsumeWithRetry запускает потребление с бесконечными ретраями при ошибках
func (c *Consumer) ConsumeWithRetry(ctx context.Context, handler func(ctx context.Context, event domain.StockReservedEvent) error) {
	c.handler = handler
	retryDelay := 3 * time.Second
	maxDelay := 30 * time.Second

	for {
		if ctx.Err() != nil {
			c.logger.Info("context cancelled, stopping consumer")
			return
		}

		err := c.consumer.Consume(ctx, []string{c.topic}, c)
		if err != nil {
			c.logger.Error("consumer error, will retry",
				zap.Error(err),
				zap.Duration("retry_delay", retryDelay),
			)
			time.Sleep(retryDelay)
			if retryDelay < maxDelay {
				retryDelay *= 2
			}
			continue
		}

		retryDelay = 3 * time.Second
	}
}

// Setup, Cleanup, ConsumeClaim — реализация sarama.ConsumerGroupHandler
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event domain.StockReservedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("unmarshal event", zap.Error(err))
			session.MarkMessage(msg, "")
			continue
		}

		if err := c.handler(session.Context(), event); err != nil {
			c.logger.Error("handle event failed", zap.Error(err))
			continue
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
