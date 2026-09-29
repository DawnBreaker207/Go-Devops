-- Counter handover mark: when walk-in staff hand pre-bought tickets over,
-- the order keeps everything and only gains a timestamp (reprints stay
-- possible; the mark only tells the next shift it was already handed over).
ALTER TABLE bookings ADD COLUMN collected_at TIMESTAMPTZ;
