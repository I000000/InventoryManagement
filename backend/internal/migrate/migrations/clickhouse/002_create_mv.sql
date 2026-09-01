-- +goose Up
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

-- +goose Down
DROP VIEW IF EXISTS daily_reservations_stats;