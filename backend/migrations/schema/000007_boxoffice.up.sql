ALTER TABLE bookings ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE bookings ADD COLUMN sold_via varchar(16) NOT NULL DEFAULT 'online';
ALTER TABLE bookings ADD COLUMN customer_name varchar(255);
ALTER TABLE bookings ADD COLUMN customer_phone varchar(20);
ALTER TABLE bookings ADD CONSTRAINT ck_booking_sold_via CHECK (sold_via IN ('online','counter'));
ALTER TABLE bookings DROP CONSTRAINT ck_booking_paid;
ALTER TABLE bookings ADD CONSTRAINT ck_booking_paid CHECK (
    (paid_at IS NULL) = (payment_id IS NULL)
    OR (sold_via = 'counter' AND paid_at IS NOT NULL AND payment_id IS NULL));