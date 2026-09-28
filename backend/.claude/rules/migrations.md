---
paths:
  - "**/migrations/**/*.sql"
---

# Migration rules

- `migrations/schema/` is the **only** source of schema truth. There is no `AutoMigrate` anywhere in this repo.
- **Pre-first-release (where this repo is today), module files ARE edited in place.** The git history is explicit
  about it (`refactor(migrations): one schema file per module, no ALTER TABLE`, `fix(review): fold ALTER
  migrations back into their modules`). The consequence is the part to remember: golang-migrate has already
  recorded that version in `schema_migrations`, so an edited file is **not** re-applied — `make migrate-up` is a
  no-op and you must `make migrate-db-reset` (destroys all data) for the change to land.
- **After the first production release this flips**: the files freeze, and every change becomes a new pair via
  `make migrate-create name=<module>_<change>` -> `migrations/schema/NNNNNN_<name>.{up,down}.sql`. Never edit a
  file that a production database has applied.
- Ask which side of that line you are on before editing anything under `migrations/schema/`.
- One six-digit sequence is shared by all modules. Current set: `000001_auth`, `000002_catalog`,
  `000003_booking_payment`, `000004_audit`, `000005_batch_report`, `000006_combo`, `000007_article`,
  `000008_notification_preferences`, `000009_promo`, `000010_pricing`.
- Pre-first-release convention: each module file holds one `CREATE TABLE` per table in its **final** shape with
  its indexes and constraints next to it — no `ALTER TABLE`, no data migration. After the first production
  release these files freeze and every change becomes a new append-only migration.
- Every `down.sql` drops that module's tables in reverse dependency order.
- Seeds are **not** tracked by golang-migrate: `migrations/seed/NNNNNN_name.sql` numbered from `010001`, applied
  in order by `make migrate-seed` through psql. They must be idempotent — fixed UUIDs plus `ON CONFLICT DO NOTHING`.
- **One deliberate exception: `010005_seed_showtimes.sql` refreshes instead of skipping.** A showtime is only
  useful while it is in the future, so "re-running changes nothing" would mean the seed stops working within
  days. It computes every start time from the cinema-local clock at run time, deletes its own previous rows and
  re-inserts them, which makes `make migrate-seed` the supported way to revive a dev database whose data has
  aged out. A seed may only do this if it (a) scopes the delete to a UUID prefix it alone owns, (b) refuses to
  delete rows other tables reference — here, showtimes with held or booked seats — and (c) skips rather than
  aborts when a row would violate a constraint, since `ex_showtime_no_hall_overlap` is an EXCLUDE constraint
  that would otherwise take the whole seed run down with it.
- A new **column** cannot be backfilled by `ON CONFLICT DO NOTHING` — existing rows are skipped entirely. Add a
  guarded `UPDATE … WHERE <column> IS NULL OR <column> = ''` beneath the INSERT, as `010001_seed_movies.sql`
  does for `backdrop_url` and `trailer_url`. Guarding on empty is what keeps it from overwriting real data.
- `make migrate-create-seed name=<name>` numbers from the **file count**, not the last number, so it emits
  `0000NN_<name>.sql` — which sorts *before* `010001` and collides next time. Rename the result to the next
  `0100NN` by hand.
- The `migrate` CLI is **not installed on this machine**. It is required by `migrate-up`, `migrate-down`,
  `migrate-create` and `migrate-db-reset`; `migrate-seed` (psql through docker compose) and `migrate-create-seed`
  (touches a file) work without it.
- `make migrate-db-reset` **destroys all data** in the dev database. Confirm with the user before running it.
- Changing a column that a model or DTO exposes means updating `internal/models/`, any `oneof=` binding tag, and
  telling the frontend — `FrontEnd-CP/src/types` is hand-written.
