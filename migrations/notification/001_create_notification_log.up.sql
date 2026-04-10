CREATE TABLE IF NOT EXISTS notification_log (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL DEFAULT 'email',
    subject VARCHAR(500) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_log_event_type ON notification_log(event_type);
CREATE INDEX idx_notification_log_created_at ON notification_log(created_at DESC);
