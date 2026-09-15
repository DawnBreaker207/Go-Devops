-- col_span lets a seat block span two columns (couple rows); the neighbor
-- column holds no separate seat. The EXCLUDE constraint is the database-level
-- backstop for two showtimes overlapping in the same hall: the service already
-- serializes on the hall row, this guards any path that forgets the lock.
ALTER TABLE seats ADD COLUMN col_span SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE seats ADD CONSTRAINT ck_seat_col_span CHECK (col_span IN (1, 2));

CREATE EXTENSION IF NOT EXISTS btree_gist;
ALTER TABLE showtimes ADD CONSTRAINT ex_showtime_no_hall_overlap
	EXCLUDE USING gist (hall_id WITH =, tstzrange(start_at, end_at) WITH &&)
	WHERE (deleted_at IS NULL);