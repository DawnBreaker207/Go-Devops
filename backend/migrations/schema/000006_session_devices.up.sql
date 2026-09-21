-- Adds per-device session tracking to refresh_tokens (GET/DELETE /users/me/sessions).
-- device_id/user_agent are carried forward across token rotations within the same
-- family so one row (the not-yet-used, not-revoked one) represents one signed-in device.

ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS device_id    VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS user_agent   VARCHAR(512),
    ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_device ON refresh_tokens (user_id, device_id);
