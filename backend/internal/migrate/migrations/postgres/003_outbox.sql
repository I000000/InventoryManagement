-- +goose Up
CREATE TABLE IF NOT EXISTS outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    traceparent TEXT,
    tracestate TEXT,
    request_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(32) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox(status, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_next_retry ON outbox(next_retry_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS outbox;