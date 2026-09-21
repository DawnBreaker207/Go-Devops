-- Widens three status enums for the customer-facing UX upgrade:
--   * movies.status gains 'coming_soon' (a movie announced but not yet on sale).
--   * showtimes.status gains 'cancelled' (the cinema actively cancelled an
--     already-sold showtime; distinct from a plain DELETE, which is refused
--     once a showtime has bookings).
--   * tickets.status gains 'void' (a ticket whose showtime was cancelled; it
--     can never be redeemed).

ALTER TABLE movies DROP CONSTRAINT IF EXISTS ck_movie_status;
ALTER TABLE movies ADD CONSTRAINT ck_movie_status CHECK (status IN ('draft','coming_soon','showing','ended'));

ALTER TABLE showtimes DROP CONSTRAINT IF EXISTS ck_showtime_status;
ALTER TABLE showtimes ADD CONSTRAINT ck_showtime_status CHECK (status IN ('open','closed','cancelled'));

ALTER TABLE tickets DROP CONSTRAINT IF EXISTS ck_ticket_status;
ALTER TABLE tickets ADD CONSTRAINT ck_ticket_status CHECK (status IN ('issued','redeemed','void'));
