# Migration layout

Two folders, each with its own tool — golang-migrate tracks ONE version number
in `schema_migrations`, so migrations in `schema/` share a single versioned
sequence.

- `schema/` — versioned schema changes, applied by golang-migrate
  (`make migrate-up` / `make migrate-down`). Files MUST be `NNNNNN_name.{up,down}.sql`
  from `make migrate-create`.
- `seed/` — plain idempotent SQL scripts, applied by `make migrate-seed`
  (runs each `NNNNNN_name.sql` in order through psql). Fixed IDs +
  `ON CONFLICT DO NOTHING` make re-runs harmless.

## Version numbers follow tracks

Each version owns one business track, so a breaking change stays reviewable in
place (tables can only be dropped by `migrate-db-reset` — running versions are
append-only):

| Version | Track                  | Tables |
|---------|------------------------|--------|
| 000001  | init/idempotent        | users, movies |
| 000002  | catalog (track 2)      | halls, seats, hall_prices, showtimes, showtime_seats |
| 000003  | booking (track 3)      | bookings, tickets |
| 000004  | operations (track 5)   | audit_logs, batch_jobs, daily_aggregates |

`migrate-create` scaffolds the next version number automatically; place a new
track's DDL in its own version instead of reusing an existing one.