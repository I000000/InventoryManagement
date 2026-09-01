package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TraceSQL выполняет функцию fn с созданием спана для SQL-операции
func TraceSQL(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	tracer := otel.Tracer("inventory-api")
	ctx, span := tracer.Start(ctx, "SQL: "+operation,
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("db.error", true))
	}
	return err
}
