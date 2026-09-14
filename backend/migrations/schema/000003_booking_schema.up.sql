-- Booking schema (Track 3): bookings and their tickets. DB-level guards:
-- at most one PENDING booking per user per showtime, and an idempotency key
-- may only ever back a single PENDING booking (final states are checked in
-- the payment flow, this index exists to stop double-create races).

CREATE TABLE IF NOT EXISTS bookings (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    showtime_id     UUID NOT NULL REFERENCES showtimes(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    total_amount    BIGINT      NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    idempotency_key VARCHAR(128),
    provider        VARCHAR(32),
    txn_ref         VARCHAR(128),
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT ck_booking_status CHECK (status IN ('pending','confirmed','expired','refunded'))
);

CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings (user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_showtime_id ON bookings (showtime_id);
CREATE INDEX IF NOT EXISTS idx_bookings_paid_at ON bookings (paid_at);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings (status);
CREATE INDEX IF NOT EXISTS idx_bookings_deleted_at ON bookings (deleted_at);

-- One PENDING booking per user per show, enforced at the DB level.
CREATE UNIQUE INDEX IF NOT EXISTS one_pending_per_user_show
    ON bookings (user_id, showtime_id) WHERE status = 'pending';

-- idempotency_key may create a PENDING booking only once; skipped for final states.
CREATE UNIQUE INDEX IF NOT EXISTS uq_bookings_idempotency_pending
    ON bookings (idempotency_key) WHERE status = 'pending' AND idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS tickets (
    id               UUID PRIMARY KEY,
    booking_id       UUID NOT NULL REFERENCES bookings(id),
    showtime_seat_id UUID NOT NULL REFERENCES showtime_seats(id),
    price            BIGINT NOT NULL CHECK (price >= 0),
    code             VARCHAR(32) NOT NULL,
    status           VARCHAR(16) NOT NULL DEFAULT 'issued',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ticket_showtime_seat UNIQUE (showtime_seat_id),
    CONSTRAINT uq_ticket_code UNIQUE (code),
    CONSTRAINT ck_ticket_status CHECK (status IN ('issued','redeemed'))
);

CREATE INDEX IF NOT EXISTS idx_tickets_booking_id ON tickets (booking_id);