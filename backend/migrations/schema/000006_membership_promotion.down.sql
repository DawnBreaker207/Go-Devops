DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_immutable();

ALTER TABLE bookings DROP COLUMN IF EXISTS membership_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS voucher_discount_amount;
ALTER TABLE bookings DROP COLUMN IF EXISTS voucher_id;

DROP TABLE IF EXISTS admin_permissions;
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS booking_combos;
DROP TABLE IF EXISTS combos;
DROP TABLE IF EXISTS user_memberships;
DROP TABLE IF EXISTS membership_tiers;
DROP TABLE IF EXISTS voucher_redemptions;
DROP TABLE IF EXISTS vouchers;

ALTER TABLE users DROP CONSTRAINT ck_user_role;
ALTER TABLE users ADD CONSTRAINT ck_user_role CHECK (role IN ('customer','staff','admin'));
