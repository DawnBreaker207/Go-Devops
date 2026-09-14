-- Module catalog: movies, halls, seats, prices, showtimes and per-show seat
-- state (F2–F6). Seats are generated from a hall layout; ticket sales only
-- touch showtime_seats.

CREATE TABLE IF NOT EXISTS movies (
    id           UUID PRIMARY KEY,
    title        VARCHAR(255) NOT NULL,
    genre        VARCHAR(100) NOT NULL,
    duration     INTEGER      NOT NULL,
    director     VARCHAR(255) NOT NULL,
    description  TEXT,
    poster_url   VARCHAR(512),
    release_date DATE         NOT NULL,
    status       VARCHAR(32)  NOT NULL DEFAULT 'draft',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    CONSTRAINT ck_movie_status CHECK (status IN ('draft','showing','ended')),
    CONSTRAINT ck_movie_duration CHECK (duration > 0)
);

CREATE INDEX IF NOT EXISTS idx_movies_status ON movies (status);

CREATE TABLE IF NOT EXISTS halls (
    id             UUID PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    rows           INTEGER      NOT NULL CHECK (rows > 0),
    seats_per_row  INTEGER      NOT NULL CHECK (seats_per_row > 0),
    seat_types     JSONB        NOT NULL DEFAULT '{}',
    gaps           JSONB        NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_halls_name ON halls (name) WHERE deleted_at IS NULL;

-- The unique key (hall_id, ...) also serves lookups by hall.
CREATE TABLE IF NOT EXISTS seats (
    id         UUID PRIMARY KEY,
    hall_id    UUID        NOT NULL REFERENCES halls(id),
    row_label  VARCHAR(8)  NOT NULL,
    col_number INTEGER     NOT NULL CHECK (col_number > 0),
    seat_type  VARCHAR(16) NOT NULL DEFAULT 'standard',
    is_gap     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_seat_hall_row_col UNIQUE (hall_id, row_label, col_number),
    CONSTRAINT ck_seat_type CHECK (seat_type IN ('standard','vip','couple','recliner'))
);

CREATE TABLE IF NOT EXISTS hall_prices (
    id         UUID PRIMARY KEY,
    hall_id    UUID        NOT NULL REFERENCES halls(id),
    seat_type  VARCHAR(16) NOT NULL,
    price      BIGINT      NOT NULL CHECK (price > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hall_prices_seat_type UNIQUE (hall_id, seat_type),
    CONSTRAINT ck_hall_prices_seat_type CHECK (seat_type IN ('standard','vip','couple','recliner'))
);

CREATE TABLE IF NOT EXISTS showtimes (
    id         UUID PRIMARY KEY,
    movie_id   UUID        NOT NULL REFERENCES movies(id),
    hall_id    UUID        NOT NULL REFERENCES halls(id),
    start_at   TIMESTAMPTZ NOT NULL,
    end_at     TIMESTAMPTZ NOT NULL,
    status     VARCHAR(16) NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_showtime_status CHECK (status IN ('open','closed')),
    CONSTRAINT ck_showtime_range CHECK (end_at > start_at)
);

CREATE INDEX IF NOT EXISTS idx_showtimes_hall_start ON showtimes (hall_id, start_at);
CREATE INDEX IF NOT EXISTS idx_showtimes_movie_start ON showtimes (movie_id, start_at);
-- Day listings, the staff board and closeDay filter by start time only.
CREATE INDEX IF NOT EXISTS idx_showtimes_start ON showtimes (start_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS showtime_seats (
    id          UUID PRIMARY KEY,
    showtime_id UUID        NOT NULL REFERENCES showtimes(id),
    seat_id     UUID        NOT NULL REFERENCES seats(id),
    status      VARCHAR(16) NOT NULL DEFAULT 'available',
    held_by     UUID,
    held_until  TIMESTAMPTZ,
    version     BIGINT      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_showtime_seat UNIQUE (showtime_id, seat_id),
    CONSTRAINT ck_showtime_seat_status CHECK (status IN ('available','held','sold')),
    CONSTRAINT ck_showtime_seat_hold CHECK ((status = 'held') = (held_by IS NOT NULL AND held_until IS NOT NULL))
);

-- Sweep: expired holds.
CREATE INDEX IF NOT EXISTS idx_showtime_seats_held ON showtime_seats (held_until) WHERE status = 'held';
