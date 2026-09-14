ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS provider VARCHAR(32),
    ADD COLUMN IF NOT EXISTS txn_ref  VARCHAR(128);

-- Keep the attempt that settled the booking, else the most recent one.
UPDATE bookings b SET provider = p.provider, txn_ref = p.txn_ref
FROM (
    SELECT DISTINCT ON (p2.booking_id) p2.booking_id, p2.provider, p2.txn_ref
    FROM payments p2
    LEFT JOIN bookings b2 ON b2.payment_id = p2.id
    ORDER BY p2.booking_id, (b2.id IS NOT NULL) DESC, p2.created_at DESC
) p
WHERE p.booking_id = b.id;

DROP INDEX IF EXISTS idx_bookings_email_pending;
DROP INDEX IF EXISTS idx_bookings_pending_expires;

ALTER TABLE bookings
    DROP COLUMN IF EXISTS payment_id,
    DROP COLUMN IF EXISTS email_sent_at,
    DROP COLUMN IF EXISTS status_reason;

DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS booking_seats;
