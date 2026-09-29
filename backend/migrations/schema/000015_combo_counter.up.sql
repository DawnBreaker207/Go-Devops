-- Counter concession sales: walk-in orders have no account, a channel and a
-- recorded takings method. Existing rows are all online customer orders.
ALTER TABLE combo_orders ALTER COLUMN user_id DROP NOT NULL;

ALTER TABLE combo_orders ADD COLUMN sold_channel VARCHAR(16) NOT NULL DEFAULT 'online';
ALTER TABLE combo_orders ADD CONSTRAINT ck_combo_orders_channel
    CHECK (sold_channel IN ('online', 'counter'));

ALTER TABLE combo_orders ADD COLUMN pay_method VARCHAR(16);
ALTER TABLE combo_orders ADD CONSTRAINT ck_combo_orders_pay_method
    CHECK (pay_method IS NULL OR pay_method IN ('cash', 'pos'));

ALTER TABLE combo_orders ADD COLUMN customer_name VARCHAR(255);

-- Counter sales land collected (handed over at the till), a state the
-- original check predates.
ALTER TABLE combo_orders DROP CONSTRAINT IF EXISTS ck_combo_order_status;
ALTER TABLE combo_orders ADD CONSTRAINT ck_combo_order_status
    CHECK (status IN ('pending', 'confirmed', 'cancelled', 'collected'));
