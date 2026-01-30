CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(255) NOT NULL, -- e.g. "OrderCreated"
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'new', -- new, processing, processed, failed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_outbox_status_created_at ON outbox(status, created_at);
