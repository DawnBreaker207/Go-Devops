-- Seed data: showtimes, so a fresh database can actually be booked against and the
-- customer home has something to show. Without this the catalogue renders but
-- `GET /showtimes` answers `[]`, the day strip on /film/:id is empty and the whole
-- booking wizard is unreachable.
--
-- WHY THIS ONE IS NOT `ON CONFLICT DO NOTHING` LIKE THE OTHERS:
-- a showtime is only useful while it is in the FUTURE. A fixed date would be stale
-- within days and the seed would stop doing its job. So every start time here is
-- computed from the CINEMA-LOCAL clock at the moment the seed runs, and re-running
-- `make migrate-seed` REFRESHES the schedule instead of skipping it. That makes this
-- file the supported way to revive a dev database whose data has aged out.
--
-- What it will NOT touch:
--   * any showtime that is not one of ours (ids outside the 3000...  prefix),
--   * any of ours that someone has already held or booked seats on - those are real
--     history, and the foreign keys from bookings/showtime_seats would refuse anyway,
--   * any slot that would collide with an existing showtime in the same hall, which
--     `ex_showtime_no_hall_overlap` enforces as an EXCLUDE constraint. A planned slot
--     that overlaps is skipped rather than aborting the whole seed.
--
-- Times are written as `<cinema local timestamp> AT TIME ZONE 'Asia/Ho_Chi_Minh'`,
-- never as a bare timestamp: the column is `timestamptz`, and the server's own zone is
-- not guaranteed to be the cinema's.

-- 1. Drop the stale copies of our own showtimes (untouched ones only).
DELETE FROM showtimes s
WHERE s.id::text LIKE '30000000-0000-0000-0000-%'
  AND NOT EXISTS (SELECT 1 FROM bookings b WHERE b.showtime_id = s.id)
  AND NOT EXISTS (SELECT 1 FROM showtime_seats ss WHERE ss.showtime_id = s.id);

-- 2. Today: relative to NOW, so there is always something left to watch today
--    whatever time the seed runs. One per hall, so they cannot overlap each other.
--
--    Rounded UP to the next whole hour, never truncated: `date_trunc('hour', now())`
--    at 20:50 would put a "future" slot at 20:45, i.e. in the past.
--
--    The WHERE below drops any of these that has spilled past midnight. Seeded late
--    enough in the evening that is all three, and then today genuinely has nothing
--    left to show - which is honest, and tomorrow is still fully scheduled by step 3.
WITH planned(n, movie_id, hall_id, start_at) AS (
    VALUES
        (1, '10000000-0000-0000-0000-000000000001'::uuid, '20000000-0000-0000-0000-000000000001'::uuid, date_trunc('hour', now()) + interval '1 hour'),
        (2, '10000000-0000-0000-0000-000000000005'::uuid, '20000000-0000-0000-0000-000000000002'::uuid, date_trunc('hour', now()) + interval '1 hour 30 minutes'),
        (3, '10000000-0000-0000-0000-000000000002'::uuid, '20000000-0000-0000-0000-000000000003'::uuid, date_trunc('hour', now()) + interval '2 hours')
)
INSERT INTO showtimes (id, movie_id, hall_id, start_at, end_at, status)
SELECT
    ('30000000-0000-0000-0000-' || lpad(p.n::text, 12, '0'))::uuid,
    p.movie_id,
    p.hall_id,
    p.start_at,
    p.start_at + make_interval(mins => m.duration),
    'open'
FROM planned p
JOIN movies m ON m.id = p.movie_id
WHERE (p.start_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date = (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date
  AND NOT EXISTS (
    SELECT 1 FROM showtimes x
    WHERE x.hall_id = p.hall_id
      AND x.deleted_at IS NULL
      AND tstzrange(x.start_at, x.end_at) && tstzrange(p.start_at, p.start_at + make_interval(mins => m.duration))
)
ON CONFLICT (id) DO NOTHING;

-- 3. The next three days: fixed cinema-local clock times. Two slots per hall per day,
--    spaced far enough apart that the longest film (128 min) cannot run into the next.
WITH today AS (
    SELECT (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AS d
),
slots(slot, hall_id, clock, movie_id) AS (
    VALUES
        (1, '20000000-0000-0000-0000-000000000001'::uuid, time '14:00', '10000000-0000-0000-0000-000000000002'::uuid),
        (2, '20000000-0000-0000-0000-000000000001'::uuid, time '19:00', '10000000-0000-0000-0000-000000000001'::uuid),
        (3, '20000000-0000-0000-0000-000000000002'::uuid, time '16:00', '10000000-0000-0000-0000-000000000004'::uuid),
        (4, '20000000-0000-0000-0000-000000000002'::uuid, time '20:30', '10000000-0000-0000-0000-000000000005'::uuid),
        (5, '20000000-0000-0000-0000-000000000003'::uuid, time '13:30', '10000000-0000-0000-0000-000000000003'::uuid),
        (6, '20000000-0000-0000-0000-000000000003'::uuid, time '18:30', '10000000-0000-0000-0000-000000000001'::uuid)
),
days(offset_days) AS (VALUES (1), (2), (3)),
planned AS (
    SELECT
        -- 10 + day*10 + slot keeps the id stable across runs and clear of the three above.
        10 + d.offset_days * 10 + s.slot AS n,
        s.movie_id,
        s.hall_id,
        ((t.d + d.offset_days + s.clock) AT TIME ZONE 'Asia/Ho_Chi_Minh') AS start_at
    FROM slots s
    CROSS JOIN days d
    CROSS JOIN today t
)
INSERT INTO showtimes (id, movie_id, hall_id, start_at, end_at, status)
SELECT
    ('30000000-0000-0000-0000-' || lpad(p.n::text, 12, '0'))::uuid,
    p.movie_id,
    p.hall_id,
    p.start_at,
    p.start_at + make_interval(mins => m.duration),
    'open'
FROM planned p
JOIN movies m ON m.id = p.movie_id
WHERE NOT EXISTS (
    SELECT 1 FROM showtimes x
    WHERE x.hall_id = p.hall_id
      AND x.deleted_at IS NULL
      AND tstzrange(x.start_at, x.end_at) && tstzrange(p.start_at, p.start_at + make_interval(mins => m.duration))
)
ON CONFLICT (id) DO NOTHING;
