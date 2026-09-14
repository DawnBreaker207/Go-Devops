-- Booking fencing + payments (Track 3 hardening).
--
-- booking_seats: which seats a booking holds, at which fencing version and at
-- which price. Confirm must still find every one of those seats HELD by the
-- booking's user at that same version; otherwise the hold was lost (swept or
-- taken over) and the booking is refunded instead of sold (R-C1, I2, I5).
--
-- payments: provider-agnostic payment attempts. The mock provider and real
-- gateways share this table and the same flow. A booking may have several
-- attempts (the customer switches provider), at most one open attempt per
-- provider; bookings.payment_id points at the attempt whose money the booking
-- carries. provider_txn_id / provider_data keep what a gateway needs later to
-- query or refund (e.g. its own transaction number and date).

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
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payments_provider_ref UNIQUE (provider, txn_ref),
    CONSTRAINT ck_payment_status CHECK (status IN ('pending','paid','failed','refund_pending','refunded'))
);

CREATE INDEX IF NOT EXISTS idx_payments_booking ON payments (booking_id);

-- One open checkout per booking per provider (a repeated pay reuses it).
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_open_attempt
    ON payments (booking_id, provider) WHERE status = 'pending';

-- Sweep: refunds the provider has not acknowledged, open attempts to reconcile.
CREATE INDEX IF NOT EXISTS idx_payments_refund_pending
    ON payments (updated_at) WHERE status = 'refund_pending';
CREATE INDEX IF NOT EXISTS idx_payments_open_created
    ON payments (created_at) WHERE status = 'pending';

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS status_reason VARCHAR(64),
    ADD COLUMN IF NOT EXISTS email_sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payment_id    UUID REFERENCES payments(id);

-- Move the single provider/txn_ref slot of bookings into payments.
INSERT INTO payments (id, booking_id, provider, txn_ref, amount, status, paid_amount,
                      paid_at, refunded_at, expires_at, created_at, updated_at)
SELECT gen_random_uuid(), b.id, b.provider, b.txn_ref, b.total_amount,
       CASE WHEN b.status = 'refunded' THEN 'refunded'
            WHEN b.paid_at IS NOT NULL THEN 'paid'
            WHEN b.status <> 'pending' THEN 'failed'
            ELSE 'pending' END,
       CASE WHEN b.paid_at IS NOT NULL THEN b.total_amount END,
       b.paid_at,
       CASE WHEN b.status = 'refunded' THEN b.updated_at END,
       b.expires_at, b.created_at, b.updated_at
FROM bookings b
WHERE b.provider IS NOT NULL AND b.txn_ref IS NOT NULL AND b.total_amount > 0;

UPDATE bookings b SET payment_id = p.id
FROM payments p
WHERE p.booking_id = b.id AND b.paid_at IS NOT NULL;

ALTER TABLE bookings
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS txn_ref;

-- Confirmed bookings predate the ticket email job: mark them as sent so the
-- first run does not mail old orders.
UPDATE bookings SET email_sent_at = updated_at
WHERE status = 'confirmed' AND email_sent_at IS NULL;

-- Expired bookings stay visible in the order history (E-O2); they used to be
-- soft-deleted.
UPDATE bookings SET deleted_at = NULL
WHERE status = 'expired' AND deleted_at IS NOT NULL;

-- Sweep: overdue PENDING bookings.
CREATE INDEX IF NOT EXISTS idx_bookings_pending_expires
    ON bookings (expires_at) WHERE status = 'pending';

-- Email job: confirmed bookings still waiting for their ticket email.
CREATE INDEX IF NOT EXISTS idx_bookings_email_pending
    ON bookings (created_at) WHERE status = 'confirmed' AND email_sent_at IS NULL;
