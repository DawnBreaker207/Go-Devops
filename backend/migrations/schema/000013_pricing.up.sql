-- Module pricing: one GLOBAL base price per seat type plus additive/percent
-- adjustment rules (day-of-week, time window or a specific date). A hall is
-- where a showtime screens; the price of a seat type is the same in every
-- hall. The hold/seatmap/pricing-page read paths all use this engine; there
-- is deliberately no per-hall price table in a fresh database.
--
-- Fresh databases seed working prices below (the same four numbers the old
-- per-hall seeds used), so booking works without an admin setting prices
-- first. PUT /admin/pricing/base changes them afterwards.

CREATE TABLE IF NOT EXISTS seat_base_prices (
    seat_type  TEXT        PRIMARY KEY,
    price      INTEGER     NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_seat_base_prices_type  CHECK (seat_type IN ('standard', 'vip', 'couple', 'recliner')),
    CONSTRAINT ck_seat_base_prices_price CHECK (price >= 0)
);

-- A pricing rule adjusts the base price when it matches. Exactly one of
-- (day_of_week, specific_date) is expected to be set for a targeted rule; a row
-- with day_of_week, start_time, end_time AND specific_date all NULL matches
-- every day and every time - see PricingService.Quote for the matching logic.
--
-- adjust_value is SIGNED ON PURPOSE: a discount rule (e.g. a "Happy Day -20k")
-- carries a negative value, a surcharge a positive one.
CREATE TABLE IF NOT EXISTS pricing_rules (
    id            UUID        PRIMARY KEY,
    name          TEXT        NOT NULL,
    -- 0 = Sunday .. 6 = Saturday (matches Go's time.Weekday and Postgres's
    -- EXTRACT(DOW FROM ...)). NULL = matches every day (unless specific_date is set).
    day_of_week   SMALLINT    NULL,
    -- NULL/NULL = matches all day.
    start_time    TIME        NULL,
    end_time      TIME        NULL,
    -- A rule scoped to one calendar date (e.g. a holiday) rather than a weekday.
    specific_date DATE        NULL,
    adjust_kind   TEXT        NOT NULL,
    adjust_value  INTEGER     NOT NULL,
    -- Higher priority applies FIRST when two active rules both match the same
    -- showtime/seat type - see PricingService.Quote for the documented tie-break.
    priority      INTEGER     NOT NULL DEFAULT 0,
    active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_pricing_rules_dow         CHECK (day_of_week IS NULL OR day_of_week BETWEEN 0 AND 6),
    CONSTRAINT ck_pricing_rules_adjust_kind CHECK (adjust_kind IN ('percent', 'fixed'))
);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_active_priority
    ON pricing_rules (active, priority DESC);

-- Working defaults (match the historical per-hall seed prices); the pricing
-- admin API owns them from here on.
INSERT INTO seat_base_prices (seat_type, price) VALUES
    ('standard', 70000),
    ('vip',      100000),
    ('couple',   160000),
    ('recliner', 130000)
ON CONFLICT (seat_type) DO NOTHING;
