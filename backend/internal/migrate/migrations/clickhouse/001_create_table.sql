-- +goose Up
CREATE TABLE IF NOT EXISTS stock_reservations (
    event_time DateTime DEFAULT now(),
    product_id String,
    quantity Int32,
    user_id String,
    request_id String,
    status String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (product_id, event_time)
SETTINGS index_granularity = 8192;

-- +goose Down
DROP TABLE IF EXISTS stock_reservations;