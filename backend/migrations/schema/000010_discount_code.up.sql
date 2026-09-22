-- Discount codes a customer can apply to a pending order before paying, and the
-- two columns on `bookings` that record which code was used and how much it took
-- off. Replaces the frontend's display-only mock voucher, which reduced the
-- number on screen while the gateway still charged full price.
--
-- WHY `bookings.total_amount` IS NOT DISCOUNTED
-- At confirm time the service asserts that the sum of the sold seats' prices
-- equals total_amount (see finalizeTx in internal/service/booking_payment.go);
-- a booking that fails that assertion cannot confirm AFTER the money has already
-- been taken. So total_amount stays the seat subtotal, discount_amount is stored
-- beside it, and the amount actually charged is (total_amount - discount_amount),
-- which is what goes into payments.amount. Every downstream money path
-- (amount-mismatch detection, refunds) already reads payments.amount, so they
-- follow automatically.

CREATE TABLE IF NOT EXISTS discount_codes (
    id            UUID PRIMARY KEY,
    -- Stored and matched UPPERCASE; the service uppercases on the way in, so
    -- "welcome10" and "WELCOME10" are the same code.
    code          VARCHAR(32)  NOT NULL,
    description   TEXT,
    -- 'percent' takes `value` percent off, capped by max_discount when set.
    -- 'amount' takes a flat `value` VND off.
    kind          VARCHAR(16)  NOT NULL,
    value         BIGINT       NOT NULL,
    -- Only meaningful for kind='percent'; NULL means uncapped.
    max_discount  BIGINT,
    -- Order subtotal required before the code applies. 0 = no minimum.
    min_order     BIGINT       NOT NULL DEFAULT 0,
    -- NULL on either side means open-ended in that direction.
    starts_at     TIMESTAMPTZ,
    ends_at       TIMESTAMPTZ,
    -- NULL means unlimited. used_count is incremented when a code is applied.
    max_uses      INT,
    used_count    INT          NOT NULL DEFAULT 0,
    active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT ck_discount_kind  CHECK (kind IN ('percent', 'amount')),
    CONSTRAINT ck_discount_value CHECK (value > 0),
    -- A percentage above 100 would be a negative price.
    CONSTRAINT ck_discount_percent_range CHECK (kind <> 'percent' OR value <= 100),
    CONSTRAINT ck_discount_max      CHECK (max_discount IS NULL OR max_discount > 0),
    CONSTRAINT ck_discount_minorder CHECK (min_order >= 0),
    CONSTRAINT ck_discount_uses     CHECK (max_uses IS NULL OR max_uses >= 0),
    CONSTRAINT ck_discount_used     CHECK (used_count >= 0),
    CONSTRAINT ck_discount_window   CHECK (starts_at IS NULL OR ends_at IS NULL OR ends_at > starts_at)
);

-- Partial unique index rather than a plain UNIQUE: a retired code is soft-deleted
-- and its string must be reusable afterwards.
CREATE UNIQUE INDEX IF NOT EXISTS uq_discount_codes_code
    ON discount_codes (code) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_discount_codes_active
    ON discount_codes (active) WHERE deleted_at IS NULL;

-- discount_amount is the VND taken off this booking; discount_code_id keeps the
-- link for reporting. ON DELETE is deliberately absent: codes are soft-deleted,
-- never removed, so the reference stays valid.
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS discount_code_id UUID REFERENCES discount_codes(id),
    ADD COLUMN IF NOT EXISTS discount_amount  BIGINT NOT NULL DEFAULT 0;

-- A discount can never exceed the subtotal, and never turn an order negative.
ALTER TABLE bookings
    ADD CONSTRAINT ck_bookings_discount_amount
    CHECK (discount_amount >= 0 AND discount_amount <= total_amount);
