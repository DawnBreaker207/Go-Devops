#!/usr/bin/env bash
# Verifies the schema migrations are a clean round trip: up -> down (all) -> up
# lands on exactly the same schema as a single up, on a real Postgres. Needs
# the postgres container from docker-compose.yml running (make docker-up, or
# docker compose up -d postgres).
set -euo pipefail
cd "$(dirname "$0")/.."

CONTAINER=cp-postgres
DB_USER=${DATABASE_USER:-postgres}
DB_A=cinema_migtest_a
DB_B=cinema_migtest_b
URL_A="postgres://${DB_USER}:postgres@localhost:5432/${DB_A}?sslmode=disable"
URL_B="postgres://${DB_USER}:postgres@localhost:5432/${DB_B}?sslmode=disable"
OUT_DIR=$(mktemp -d)
trap 'rm -rf "$OUT_DIR"' EXIT

psql() { docker exec -i "$CONTAINER" psql -U "$DB_USER" -v ON_ERROR_STOP=1 "$@"; }
dump_schema() {
	# Drop the golang-migrate bookkeeping table before comparing: it's tooling
	# state, not application schema, and up-only vs. up-down-up leave it in
	# different (both correct) shapes.
	docker exec "$CONTAINER" pg_dump -U "$DB_USER" --schema-only --no-owner --no-privileges "$1" \
		| grep -v -E '^\s*--' \
		| grep -v 'schema_migrations' \
		| grep -v -E '^\\(un)?restrict ' \
		| sed '/^$/N;/^\n$/D' >"$2"
}

echo "== dropping any leftover scratch databases"
psql -c "DROP DATABASE IF EXISTS ${DB_A} WITH (FORCE);"
psql -c "DROP DATABASE IF EXISTS ${DB_B} WITH (FORCE);"

echo "== ${DB_A}: single up"
psql -c "CREATE DATABASE ${DB_A};"
migrate -path migrations/schema -database "$URL_A" up
dump_schema "$DB_A" "$OUT_DIR/a.sql"

echo "== ${DB_B}: up, down (all), up again"
psql -c "CREATE DATABASE ${DB_B};"
migrate -path migrations/schema -database "$URL_B" up
migrate -path migrations/schema -database "$URL_B" down -all
migrate -path migrations/schema -database "$URL_B" up
dump_schema "$DB_B" "$OUT_DIR/b.sql"

echo "== cleaning up"
psql -c "DROP DATABASE IF EXISTS ${DB_A} WITH (FORCE);"
psql -c "DROP DATABASE IF EXISTS ${DB_B} WITH (FORCE);"

echo "== diffing schema-only dumps"
if ! diff -u "$OUT_DIR/a.sql" "$OUT_DIR/b.sql"; then
	echo "FAIL: up -> down -> up did not reproduce the same schema as a single up" >&2
	exit 1
fi
echo "OK: migrations round-trip cleanly"
