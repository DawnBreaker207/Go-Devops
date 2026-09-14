-- Module booking & payment: bookings, the seats they hold, tickets and
-- payment attempts (F7–F10, F14).
--
-- booking_seats: which seats a booking holds, at which fencing version and at
-- which price. Confirm must still find every one of those seats HELD by the
-- booking's user at that same version; otherwise the hold was lost and the
-- booking is refunded instead of sold.
--
-- payments: provider-agnostic payment attempts, several per booking. A booking
-- carries the money of exactly one of them (payment_id); payments.booking_id
-- holds the foreign key, bookings.payment_id is only unique, since a cycle of
-- foreign keys would need ALTER.

CREATE TABLE IF NOT EXISTS bookings (
    id              UUID PRIMARY KEY,
    user_id         UUID        NOT NULL REFERENCES users(id),
    showtime_id     UUID        NOT NULL REFERENCES showtimes(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    status_reason   VARCHAR(64),
    total_amount    BIGINT      NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    expires_at      TIMESTAMPTZ,
    idempotency_key VARCHAR(128),
    payment_id      UUID,
    paid_at         TIMESTAMPTZ,
    -- A paid booking the sweep could not settle is retried with backoff.
    finalize_attempts INTEGER   NOT NULL DEFAULT 0 CHECK (finalize_attempts >= 0),
    next_finalize_at  TIMESTAMPTZ,
    email_sent_at   TIMESTAMPTZ,
    -- Ticket email sending is leased and retried with backoff, 6 tries at most.
    email_attempts      INTEGER NOT NULL DEFAULT 0 CHECK (email_attempts >= 0),
    email_claimed_until TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_booking_status CHECK (status IN ('pending','confirmed','expired','refunded')),
    CONSTRAINT ck_booking_paid CHECK ((paid_at IS NULL) = (payment_id IS NULL)),
    CONSTRAINT ck_booking_settled_paid CHECK (status NOT IN ('confirmed','refunded') OR paid_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_bookings_user_created ON bookings (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bookings_showtime_id ON bookings (showtime_id);
CREATE INDEX IF NOT EXISTS idx_bookings_paid_at ON bookings (paid_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bookings_payment_id ON bookings (payment_id);

-- One PENDING booking per user per show.
CREATE UNIQUE INDEX IF NOT EXISTS one_pending_per_user_show
    ON bookings (user_id, showtime_id) WHERE status = 'pending';

-- An idempotency key backs one PENDING booking; it is also looked up across
-- every status, so a reused key can be refused.
CREATE UNIQUE INDEX IF NOT EXISTS uq_bookings_idempotency_pending
    ON bookings (idempotency_key) WHERE status = 'pending' AND idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_idempotency_key
    ON bookings (idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Sweep: overdue PENDING bookings, and paid ones still PENDING.
CREATE INDEX IF NOT EXISTS idx_bookings_pending_expires
    ON bookings (expires_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_bookings_pending_paid
    ON bookings (paid_at) WHERE status = 'pending' AND paid_at IS NOT NULL;

-- Email job: ticket emails still due (given-up ones drop out).
CREATE INDEX IF NOT EXISTS idx_bookings_email_pending
    ON bookings (created_at) WHERE status = 'confirmed' AND email_sent_at IS NULL AND email_attempts < 6;

CREATE TABLE IF NOT EXISTS booking_seats (
    id               UUID PRIMARY KEY,
    booking_id       UUID        NOT NULL REFERENCES bookings(id),
    showtime_seat_id UUID        NOT NULL REFERENCES showtime_seats(id),
    seat_type        VARCHAR(16) NOT NULL,
    price            BIGINT      NOT NULL CHECK (price > 0),
    hold_version     BIGINT      NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_booking_seat UNIQUE (booking_id, showtime_seat_id)
);

CREATE INDEX IF NOT EXISTS idx_booking_seats_showtime_seat ON booking_seats (showtime_seat_id);

CREATE TABLE IF NOT EXISTS tickets (
    id               UUID PRIMARY KEY,
    booking_id       UUID        NOT NULL REFERENCES bookings(id),
    showtime_seat_id UUID        NOT NULL REFERENCES showtime_seats(id),
    price            BIGINT      NOT NULL CHECK (price >= 0),
    code             VARCHAR(32) NOT NULL,
    status           VARCHAR(16) NOT NULL DEFAULT 'issued',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ticket_showtime_seat UNIQUE (showtime_seat_id),
    CONSTRAINT uq_ticket_code UNIQUE (code),
    CONSTRAINT ck_ticket_status CHECK (status IN ('issued','redeemed'))
);

CREATE INDEX IF NOT EXISTS idx_tickets_booking_id ON tickets (booking_id);

CREATE TABLE IF NOT EXISTS payments (
    id              UUID PRIMARY KEY,
    booking_id      UUID         NOT NULL REFERENCES bookings(id),
    provider        VARCHAR(32)  NOT NULL,
    txn_ref         VARCHAR(64)  NOT NULL,
    amount          BIGINT       NOT NULL CHECK (amount > 0),
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending',
    status_reason   VARCHAR(64),
    redirect_url    TEXT,
    provider_txn_id VARCHAR(128),
    provider_data   JSONB        NOT NULL DEFAULT '{}',
    paid_amount     BIGINT,
    paid_at         TIMESTAMPTZ,
    refunded_at     TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    checked_at      TIMESTAMPTZ,
    -- Provider refunds are claimed before the call and retried with backoff.
    refund_attempts INTEGER      NOT NULL DEFAULT 0 CHECK (refund_attempts >= 0),
    next_retry_at   TIMESTAMPTZ,
    last_error      VARCHAR(512),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payments_provider_ref UNIQUE (provider, txn_ref),
    CONSTRAINT ck_payment_status CHECK (status IN ('pending','paid','failed','refund_pending','refunded')),
    CONSTRAINT ck_payment_captured CHECK (status NOT IN ('paid','refund_pending','refunded')
        OR (paid_at IS NOT NULL AND paid_amount IS NOT NULL)),
    CONSTRAINT ck_payment_refunded CHECK (status <> 'refunded' OR refunded_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_payments_booking ON payments (booking_id);

-- One open checkout per booking per provider (a repeated pay reuses it).
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_open_attempt
    ON payments (booking_id, provider) WHERE status = 'pending';

-- A booking keeps the money of one attempt; any other capture is refunded.
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_one_paid
    ON payments (booking_id) WHERE status = 'paid';

-- Sweep: refunds due for a (re)try, open attempts to reconcile, and given-up
-- attempts rechecked for money collected late.
CREATE INDEX IF NOT EXISTS idx_payments_refund_due
    ON payments (next_retry_at NULLS FIRST) WHERE status = 'refund_pending';
CREATE INDEX IF NOT EXISTS idx_payments_open_created
    ON payments (created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_payments_failed_recheck
    ON payments ((COALESCE(expires_at, created_at))) WHERE status = 'failed';
