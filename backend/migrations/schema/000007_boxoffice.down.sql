ALTER TABLE bookings DROP CONSTRAINT ck_booking_paid;
ALTER TABLE bookings DROP CONSTRAINT ck_booking_sold_via;
ALTER TABLE bookings DROP COLUMN customer_phone;
ALTER TABLE bookings DROP COLUMN customer_name;
ALTER TABLE bookings DROP COLUMN sold_via;
ALTER TABLE bookings ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE bookings ADD CONSTRAINT ck_booking_paid CHECK ((paid_at IS NULL) = (payment_id IS NULL));