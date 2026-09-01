package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TraceKafkaProduce создаёт спан для отправки сообщения в Kafka
func TraceKafkaProduce(ctx context.Context, topic string, key string) (context.Context, trace.Span) {
	tracer := otel.Tracer("inventory-api")
	ctx, span := tracer.Start(ctx, "Kafka: produce",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.message.key", key),
		),
	)
	return ctx, span
}

// TraceKafkaConsume создаёт спан для получения сообщения из Kafka
func TraceKafkaConsume(ctx context.Context, topic string, key string) (context.Context, trace.Span) {
	tracer := otel.Tracer("stock-worker")
	ctx, span := tracer.Start(ctx, "Kafka: consume",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.message.key", key),
		),
	)
	return ctx, span
}
