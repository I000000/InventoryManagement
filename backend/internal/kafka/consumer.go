package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

type Consumer struct {
	consumer    sarama.ConsumerGroup
	topic       string
	logger      *zap.Logger
	handler     func(ctx context.Context, event domain.StockReservedEvent) error
	workerCount int
}

func NewConsumer(brokers []string, groupID string, topic string, logger *zap.Logger, workerCount ...int) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Enable = false

	consumer, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	wc := 1
	if len(workerCount) > 0 && workerCount[0] > 0 {
		wc = workerCount[0]
	}

	return &Consumer{
		consumer:    consumer,
		topic:       topic,
		logger:      logger,
		workerCount: wc,
	}, nil
}

// ConsumeWithRetry запускает потребление с бесконечными ретраями при ошибках.
func (c *Consumer) ConsumeWithRetry(ctx context.Context, handler func(ctx context.Context, event domain.StockReservedEvent) error) error {
	c.handler = handler
	retryDelay := 3 * time.Second
	maxDelay := 30 * time.Second

	for {
		if ctx.Err() != nil {
			c.logger.Info("context cancelled, stopping consumer")
			return ctx.Err()
		}

		err := c.consumer.Consume(ctx, []string{c.topic}, c)
		if err != nil {
			c.logger.Error("consumer error, will retry",
				zap.Error(err),
				zap.Duration("retry_delay", retryDelay),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
			if retryDelay < maxDelay {
				retryDelay *= 2
			}
			continue
		}

		retryDelay = 3 * time.Second
	}
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Канал для сообщений (буферизированный)
	msgChan := make(chan *sarama.ConsumerMessage, c.workerCount*2)
	var wg sync.WaitGroup

	// Запускаем воркеров
	for i := 0; i < c.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range msgChan {
				c.processMessage(session, msg)
			}
		}()
	}

	// Читаем сообщения и отправляем в канал
	for msg := range claim.Messages() {
		msgChan <- msg
	}

	// Закрываем канал и ждём завершения воркеров
	close(msgChan)
	wg.Wait()
	return nil
}

func (c *Consumer) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	// Извлекаем контекст из заголовков
	carrier := propagation.MapCarrier{}
	for _, header := range msg.Headers {
		carrier[string(header.Key)] = string(header.Value)
	}
	propagator := otel.GetTextMapPropagator()
	ctx := propagator.Extract(session.Context(), carrier)

	var event domain.StockReservedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error("unmarshal event", zap.Error(err))
		session.MarkMessage(msg, "")
		return
	}

	if err := c.handler(ctx, event); err != nil {
		c.logger.Error("handle event failed", zap.Error(err))
		// Не коммитим сообщение при ошибке — оно будет обработано повторно
		return
	}

	session.MarkMessage(msg, "")
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
