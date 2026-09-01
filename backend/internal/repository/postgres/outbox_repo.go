package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type OutboxEvent struct {
	ID          int64           `db:"id"`
	EventID     string          `db:"event_id"`
	EventType   string          `db:"event_type"`
	Payload     json.RawMessage `db:"payload"`
	TraceParent string          `db:"traceparent"`
	TraceState  string          `db:"tracestate"`
	CreatedAt   time.Time       `db:"created_at"`
	ProcessedAt *time.Time      `db:"processed_at"`
	Status      string          `db:"status"`
	RetryCount  int             `db:"retry_count"`
	LastError   *string         `db:"last_error"`
	NextRetryAt *time.Time      `db:"next_retry_at"`
}

type OutboxRepository struct {
	db *sqlx.DB
}

func NewOutboxRepository(db *sqlx.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Insert(ctx context.Context, tx *sqlx.Tx, eventID, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Извлекаем контекст трассировки
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	traceparent := carrier["traceparent"]
	tracestate := carrier["tracestate"]

	query := `INSERT INTO outbox (event_id, event_type, payload, status, traceparent, tracestate)
              VALUES ($1, $2, $3, 'pending', $4, $5)`
	_, err = tx.ExecContext(ctx, query, eventID, eventType, data, traceparent, tracestate)
	return err
}

func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]OutboxEvent, error) {
	query := `SELECT id, event_id, event_type, payload, created_at, processed_at, status, retry_count, last_error, next_retry_at, traceparent, tracestate
              FROM outbox
              WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
              ORDER BY created_at ASC
              LIMIT $1`
	var events []OutboxEvent
	err := r.db.SelectContext(ctx, &events, query, limit)
	return events, err
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET status = 'processed', processed_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	query := `UPDATE outbox SET status = 'failed', retry_count = retry_count + 1, last_error = $1, next_retry_at = NOW() + (POW(2, retry_count) || ' seconds')::interval WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, errMsg, id)
	return err
}
