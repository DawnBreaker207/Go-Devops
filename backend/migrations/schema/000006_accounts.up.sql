-- Accounts (F18 minimal) + refresh token rotation (F1 E-C4).
--
-- users.active: a locked account can not log in, refresh, hold seats or pay;
-- tickets it already bought still pass the gate (E-U3).
--
-- refresh_tokens: every issued refresh token (its jti). A refresh consumes the
-- token once and issues the next one in the same family; a used token coming
-- back is a replay and revokes the whole family, so everyone holding it has to
-- log in again (T13).

ALTER TABLE users ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;

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
