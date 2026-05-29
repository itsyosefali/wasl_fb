ALTER TABLE events ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'facebook';
ALTER TABLE events ADD COLUMN IF NOT EXISTS aggregate_type TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN IF NOT EXISTS aggregate_id TEXT NOT NULL DEFAULT '';

ALTER TABLE messages ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id);

CREATE INDEX IF NOT EXISTS idx_events_channel ON events(channel);
CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(tenant_id, aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_messages_event_id ON messages(event_id);
CREATE INDEX IF NOT EXISTS idx_comments_event_id ON comments(event_id);

-- Materialized read views (events remain source of truth)
CREATE OR REPLACE VIEW messages_view AS
SELECT
    e.id,
    e.tenant_id,
    e.payload->>'external_id' AS external_id,
    (e.payload->>'page_id')::uuid AS page_id,
    (e.payload->>'contact_id')::uuid AS contact_id,
    e.payload->>'direction' AS direction,
    COALESCE(e.payload->>'text', e.payload->>'message', '') AS message,
    e.channel,
    e.created_at
FROM events e
WHERE e.event_type IN ('message.received', 'message.sent', 'instagram.message.received')
  AND e.payload ? 'page_id';

CREATE OR REPLACE VIEW comments_view AS
SELECT
    e.id,
    e.tenant_id,
    e.payload->>'external_id' AS external_id,
    (e.payload->>'page_id')::uuid AS page_id,
    (e.payload->>'contact_id')::uuid AS contact_id,
    COALESCE(e.payload->>'text', e.payload->>'message', '') AS message,
    COALESCE(e.payload->>'status', 'visible') AS status,
    e.channel,
    e.created_at
FROM events e
WHERE e.event_type IN ('comment.created', 'comment.updated', 'comment.deleted', 'comment.replied', 'instagram.comment.created')
  AND e.payload ? 'page_id';
