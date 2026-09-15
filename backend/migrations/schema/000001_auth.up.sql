-- Module auth: accounts and refresh token rotation (F1, F18).
--
-- users.active: a locked account can not log in, refresh, hold seats or pay;
-- tickets it already bought still pass the gate (E-U3).
--
-- refresh_tokens: every issued refresh token (its jti). A refresh consumes the
-- token once and issues the next one in the same family; a used token coming
-- back is a replay and revokes the whole family, so everyone holding it has to
-- log in again (T13).

CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    full_name  VARCHAR(255) NOT NULL,
    phone      VARCHAR(20),
    role       VARCHAR(32)  NOT NULL DEFAULT 'customer',
    active     BOOLEAN      NOT NULL DEFAULT TRUE,
    -- Version of the terms the account holder last accepted (0 = none yet, ND13).
    accepted_terms_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_user_role CHECK (role IN ('customer','staff','admin'))
);

-- A deleted account does not keep its email taken.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id),
    family_id  UUID        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens (expires_at);

-- Only the SHA-256 of an emailed reset token is stored; a token works once.
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id),
    token_hash CHAR(64)    NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_password_reset_token_hash UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON password_reset_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires ON password_reset_tokens (expires_at);
