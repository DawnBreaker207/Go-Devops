CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    full_name  VARCHAR(255) NOT NULL,
    role       VARCHAR(32)  NOT NULL DEFAULT 'customer',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

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
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_movies_title ON movies (title);
CREATE INDEX IF NOT EXISTS idx_movies_status ON movies (status);
CREATE INDEX IF NOT EXISTS idx_movies_deleted_at ON movies (deleted_at);
