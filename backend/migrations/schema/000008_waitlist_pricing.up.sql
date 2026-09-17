-- Module waitlist & dynamic pricing rules.
CREATE TABLE IF NOT EXISTS waitlist_entries (
    id                 UUID         PRIMARY KEY,
    user_id            UUID         NOT NULL REFERENCES users(id),
    showtime_id        UUID         NOT NULL REFERENCES showtimes(id),
    status             VARCHAR(16)  NOT NULL DEFAULT 'waiting',
    fulfilled_booking_id UUID REFERENCES bookings(id),
    notified_at        TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_waitlist_status CHECK (status IN ('waiting','notified','fulfilled','expired','canceled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS one_active_waitlist_per_user_show
    ON waitlist_entries (user_id, showtime_id) WHERE status IN ('waiting','notified');
CREATE INDEX IF NOT EXISTS idx_waitlist_showtime_status_created
    ON waitlist_entries (showtime_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_waitlist_user ON waitlist_entries (user_id);

-- Matching rules stack additively; percent is always computed on the
-- hall_prices base, so rule order does not change the outcome.
CREATE TABLE IF NOT EXISTS pricing_rules (
    id               UUID        PRIMARY KEY,
    name             VARCHAR(255) NOT NULL,
    -- 0=Sunday..6=Saturday, NULL = every day.
    day_of_week      SMALLINT,
    -- Local time-of-day window [starts_at, ends_at); NULL+NULL = all day.
    starts_at        TIME,
    ends_at          TIME,
    adjustment_type  VARCHAR(16) NOT NULL,
    adjustment_value NUMERIC(10,2) NOT NULL,
    active           BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_pricing_rule_dow CHECK (day_of_week IS NULL OR day_of_week BETWEEN 0 AND 6),
    CONSTRAINT ck_pricing_rule_window CHECK ((starts_at IS NULL) = (ends_at IS NULL)),
    CONSTRAINT ck_pricing_rule_type CHECK (adjustment_type IN ('percent','fixed'))
);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_active ON pricing_rules (active);
