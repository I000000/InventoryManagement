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

CREATE MATERIALIZED VIEW IF NOT EXISTS daily_reservations_stats
ENGINE = SummingMergeTree()
ORDER BY (product_id, date)
AS SELECT
    product_id,
    toDate(event_time) AS date,
    sum(quantity) AS total_reserved,
    count() AS total_operations
FROM stock_reservations
GROUP BY product_id, date;