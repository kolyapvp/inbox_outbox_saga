-- Adds saga choreography demo tables/columns.
-- NOTE: These init scripts are executed only on a fresh Postgres volume.

-- Enrich orders with ticket details (optional fields for demo UI)
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS from_city TEXT,
  ADD COLUMN IF NOT EXISTS to_city TEXT,
  ADD COLUMN IF NOT EXISTS travel_date DATE,
  ADD COLUMN IF NOT EXISTS travel_time TEXT,
  ADD COLUMN IF NOT EXISTS airline TEXT;

CREATE INDEX IF NOT EXISTS idx_orders_route_date ON orders(from_city, to_city, travel_date);

-- Outbox metadata (correlation/causation for saga tracing)
ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS correlation_id UUID,
  ADD COLUMN IF NOT EXISTS causation_id UUID,
  ADD COLUMN IF NOT EXISTS producer TEXT NOT NULL DEFAULT 'unknown';

CREATE INDEX IF NOT EXISTS idx_outbox_correlation_created_at ON outbox(correlation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_producer_created_at ON outbox(producer, created_at);

-- Inbox (consumer-side deduplication) with per-consumer keys
CREATE TABLE IF NOT EXISTS inbox_events (
  consumer TEXT NOT NULL,
  event_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  correlation_id UUID,
  processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (consumer, event_id)
);

CREATE INDEX IF NOT EXISTS idx_inbox_correlation_processed_at ON inbox_events(correlation_id, processed_at);
CREATE INDEX IF NOT EXISTS idx_inbox_consumer_processed_at ON inbox_events(consumer, processed_at);

-- Payment service local state
CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL,
  amount DECIMAL(10, 2) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);

-- Ticket service local state
CREATE TABLE IF NOT EXISTS tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(id),
  from_city TEXT,
  to_city TEXT,
  travel_date DATE,
  travel_time TEXT,
  airline TEXT,
  status TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tickets_order_id ON tickets(order_id);
