# Migration layout

Two folders, each with its own tool — golang-migrate tracks ONE version number
in `schema_migrations`, so migrations in `schema/` share a single versioned
sequence.

- `schema/` — the database schema, applied by golang-migrate
  (`make migrate-up` / `make migrate-down`). Files are `NNNNNN_module.{up,down}.sql`.
  It is the only source of the schema: the server never runs GORM AutoMigrate.
- `seed/` — plain idempotent SQL scripts, applied by `make migrate-seed`
  (runs each `NNNNNN_name.sql` in order through psql). Fixed IDs +
  `ON CONFLICT DO NOTHING` make re-runs harmless.

## One file per module

| Version | Module            | Tables |
|---------|-------------------|--------|
| 000001  | auth              | users, refresh_tokens |
| 000002  | catalog           | movies, halls, seats, hall_prices, showtimes, showtime_seats |
| 000003  | booking & payment | bookings, booking_seats, tickets, payments |
| 000004  | audit             | audit_logs |
| 000005  | batch & report    | batch_jobs, daily_aggregates |

Rules:

- Each table has exactly one `CREATE TABLE` holding its final shape, in the file
  of its module, with its indexes and constraints next to it. No `ALTER TABLE`,
  no data migration.
- `bookings.payment_id` and `payments.booking_id` point at each other; only
  `payments.booking_id` is a foreign key (a cycle would need `ALTER`),
  `bookings.payment_id` is indexed.
- `down.sql` drops the module's tables in reverse dependency order.

## Changing the schema

**Until the first production release:** edit the `CREATE TABLE` in the module
file directly, then rebuild the dev database with `make migrate-db-reset`
(drops `cinema`, applies every migration, runs the seeds). Tests build their own
database from these files on every run.

**After the first production release:** real data must survive, so the files
above become frozen. Every change is a new, append-only migration
(`make migrate-create name=<module>_<change>`), which may then use `ALTER TABLE`.
