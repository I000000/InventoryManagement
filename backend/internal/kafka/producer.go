package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

type kafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	logger   *zap.Logger
}

var _ service.EventProducer = (*kafkaProducer)(nil)

func NewProducer(brokers []string, topic string, logger *zap.Logger) (service.EventProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("create producer: %w", err)
	}

	return &kafkaProducer{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}, nil
}

func (p *kafkaProducer) SendStockReservedEvent(ctx context.Context, event domain.StockReservedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.ProductID),
		Value: sarama.ByteEncoder(data),
	}

	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	for k, v := range carrier {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("failed to send Kafka message",
			zap.String("product_id", event.ProductID),
			zap.Error(err),
		)
		return fmt.Errorf("send message: %w", err)
	}
	p.logger.Debug("sent Kafka message",
		zap.String("product_id", event.ProductID),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)
	return nil
}

func (p *kafkaProducer) Close() error {
	return p.producer.Close()
}
