CREATE TABLE IF NOT EXISTS stocks (
    id SERIAL PRIMARY KEY,
    product_id VARCHAR(64) UNIQUE NOT NULL,
    total_count INT NOT NULL CHECK (total_count >= 0),
    reserved_count INT NOT NULL DEFAULT 0 CHECK (reserved_count >= 0),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stocks_product_id ON stocks(product_id);

INSERT INTO stocks (product_id, total_count) VALUES 
    ('iphone_15_pro', 1000000),
    ('iphone_15', 50000000),
    ('samsung_s24_ultra', 800000),
    ('samsung_s24', 1200000),
    ('sony_wh1000xm5', 300000),
    ('apple_watch_series9', 700000),
    ('macbook_air_m2', 400000),
    ('macbook_pro_m3', 200000),
    ('ipad_pro_12_9', 600000),
    ('airpods_pro_2', 1500000)
ON CONFLICT (product_id) DO NOTHING;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_stocks_updated_at ON stocks;
CREATE TRIGGER update_stocks_updated_at
    BEFORE UPDATE ON stocks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();