-- Walk-in rows keep working after down: user_id stays nullable (re-adding
-- NOT NULL would fail while counter orders exist), added columns just drop.
-- Restoring the old status check fails while collected rows exist: collect
-- (refund/void them first) or keep the migration applied.
ALTER TABLE combo_orders DROP CONSTRAINT IF EXISTS ck_combo_order_status;
ALTER TABLE combo_orders ADD CONSTRAINT ck_combo_order_status
    CHECK (status IN ('pending', 'confirmed', 'cancelled'));
ALTER TABLE combo_orders DROP CONSTRAINT IF EXISTS ck_combo_orders_pay_method;
ALTER TABLE combo_orders DROP CONSTRAINT IF EXISTS ck_combo_orders_channel;
ALTER TABLE combo_orders DROP COLUMN IF EXISTS customer_name;
ALTER TABLE combo_orders DROP COLUMN IF EXISTS pay_method;
ALTER TABLE combo_orders DROP COLUMN IF EXISTS sold_channel;
