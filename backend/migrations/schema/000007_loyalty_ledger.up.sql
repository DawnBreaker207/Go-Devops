-- Module loyalty: points/rewards, a unified money ledger, membership
-- reminder tracking.

ALTER TABLE user_memberships ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ;

ALTER TABLE vouchers ALTER COLUMN created_by_admin_id DROP NOT NULL;
ALTER TABLE vouchers ADD COLUMN IF NOT EXISTS is_system_issued BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE vouchers ADD COLUMN IF NOT EXISTS assigned_user_id UUID REFERENCES users(id);
CREATE INDEX IF NOT EXISTS idx_vouchers_assigned_user ON vouchers (assigned_user_id) WHERE assigned_user_id IS NOT NULL;

-- balance is CAS-updated (WHERE balance >= cost); point_transactions is the
-- append-only log, never summed for the live balance.
CREATE TABLE IF NOT EXISTS user_points (
    user_id                  UUID PRIMARY KEY REFERENCES users(id),
    balance                  BIGINT      NOT NULL DEFAULT 0,
    tickets_toward_milestone INTEGER     NOT NULL DEFAULT 0,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_user_points_balance CHECK (balance >= 0),
    CONSTRAINT ck_user_points_milestone CHECK (tickets_toward_milestone >= 0)
);

CREATE TABLE IF NOT EXISTS point_transactions (
    id                   UUID        PRIMARY KEY,
    user_id              UUID        NOT NULL REFERENCES users(id),
    amount                BIGINT     NOT NULL,
    type                 VARCHAR(16) NOT NULL,
    reference_booking_id UUID REFERENCES bookings(id),
    reference_reward_id  UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_point_tx_type CHECK (type IN ('earn','redeem','refund_reversal'))
);

CREATE INDEX IF NOT EXISTS idx_point_transactions_user ON point_transactions (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_point_transactions_booking ON point_transactions (reference_booking_id) WHERE reference_booking_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS rewards (
    id                  UUID         PRIMARY KEY,
    type                VARCHAR(16)  NOT NULL,
    name                VARCHAR(255) NOT NULL,
    points_cost         BIGINT       NOT NULL,
    stock_quantity      INTEGER,
    voucher_template_id UUID REFERENCES vouchers(id),
    active              BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_reward_type CHECK (type IN ('gift','voucher')),
    CONSTRAINT ck_reward_points_cost CHECK (points_cost > 0),
    CONSTRAINT ck_reward_stock CHECK (stock_quantity IS NULL OR stock_quantity >= 0)
);

CREATE TABLE IF NOT EXISTS reward_redemptions (
    id                UUID         PRIMARY KEY,
    reward_id         UUID         NOT NULL REFERENCES rewards(id),
    user_id           UUID         NOT NULL REFERENCES users(id),
    points_spent      BIGINT       NOT NULL,
    code              VARCHAR(32)  NOT NULL,
    status            VARCHAR(16)  NOT NULL DEFAULT 'pending_pickup',
    issued_voucher_id UUID REFERENCES vouchers(id),
    delivered_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_reward_redemption_status CHECK (status IN ('pending_pickup','delivered')),
    CONSTRAINT uq_reward_redemptions_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_reward_redemptions_user ON reward_redemptions (user_id);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id              UUID         PRIMARY KEY,
    type            VARCHAR(32)  NOT NULL,
    amount          BIGINT       NOT NULL,
    reference_table VARCHAR(32)  NOT NULL,
    reference_id    UUID         NOT NULL,
    user_id         UUID REFERENCES users(id),
    booking_id      UUID REFERENCES bookings(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_ledger_entry_type CHECK (type IN ('payment_captured','payment_refunded','membership_purchase','reward_redeemed'))
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_type_created ON ledger_entries (type, created_at);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_booking ON ledger_entries (booking_id) WHERE booking_id IS NOT NULL;

-- Tamper-evident hash chaining for audit_logs was tried and reverted: the
-- serializing lock deadlocked with existing seat/booking locks.
