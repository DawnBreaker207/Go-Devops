DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS reward_redemptions;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS point_transactions;
DROP TABLE IF EXISTS user_points;

ALTER TABLE vouchers DROP COLUMN IF EXISTS assigned_user_id;
ALTER TABLE vouchers DROP COLUMN IF EXISTS is_system_issued;
ALTER TABLE vouchers ALTER COLUMN created_by_admin_id SET NOT NULL;

ALTER TABLE user_memberships DROP COLUMN IF EXISTS reminder_sent_at;
