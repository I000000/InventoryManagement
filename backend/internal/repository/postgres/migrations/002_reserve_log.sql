CREATE TABLE IF NOT EXISTS reserve_log (
    id SERIAL PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    quantity INT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64),
    status VARCHAR(32) NOT NULL, -- 'success', 'failed', 'duplicate'
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reserve_log_request_id ON reserve_log(request_id);
CREATE INDEX IF NOT EXISTS idx_reserve_log_product_id ON reserve_log(product_id);
CREATE INDEX IF NOT EXISTS idx_reserve_log_created_at ON reserve_log(created_at);