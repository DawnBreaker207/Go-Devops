-- Per-user notification opt-in flags: booking/schedule reminders vs promo/
-- marketing alerts (F4). A user may turn both off; nothing here forces at
-- least one category to stay on. One row per user, created lazily on first
-- GET /users/me/notification-preferences.

CREATE TABLE IF NOT EXISTS notification_preferences (
    id                 UUID PRIMARY KEY,
    user_id            UUID        NOT NULL UNIQUE REFERENCES users(id),
    booking_reminders  BOOLEAN     NOT NULL DEFAULT TRUE,
    promo_offers       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
