ALTER TABLE tickets DROP CONSTRAINT IF EXISTS ck_ticket_status;
ALTER TABLE tickets ADD CONSTRAINT ck_ticket_status CHECK (status IN ('issued','redeemed'));

ALTER TABLE showtimes DROP CONSTRAINT IF EXISTS ck_showtime_status;
ALTER TABLE showtimes ADD CONSTRAINT ck_showtime_status CHECK (status IN ('open','closed'));

ALTER TABLE movies DROP CONSTRAINT IF EXISTS ck_movie_status;
ALTER TABLE movies ADD CONSTRAINT ck_movie_status CHECK (status IN ('draft','showing','ended'));
