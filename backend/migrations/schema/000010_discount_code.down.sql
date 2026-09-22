-- Drops the constraint before the columns it references, and the bookings
-- columns before the table they point at.
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS ck_bookings_discount_amount;

ALTER TABLE bookings
    DROP COLUMN IF EXISTS discount_code_id,
    DROP COLUMN IF EXISTS discount_amount;

DROP INDEX IF EXISTS idx_discount_codes_active;
DROP INDEX IF EXISTS uq_discount_codes_code;
DROP TABLE IF EXISTS discount_codes;
