DROP INDEX IF EXISTS idx_refresh_tokens_user_device;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS user_agent,
    DROP COLUMN IF EXISTS device_id;
