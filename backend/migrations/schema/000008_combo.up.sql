-- Module combo: concession/combo products sold independently of a ticket
-- booking (F5). combo_orders.booking_id is an optional correlation only — no
-- write here ever runs inside a booking transaction, and a combo order failing
-- must never roll back or block a booking.

-- Named "concession_items", not "combos": this database already has an
-- unrelated, more advanced "combos" table (member pricing, per-branch stock,
-- referenced by booking_combos/combo_branch_stock) from a different
-- in-progress feature outside this migration set. Renamed to avoid colliding
-- with it.
CREATE TABLE IF NOT EXISTS concession_items (
    id          UUID PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    price       BIGINT       NOT NULL CHECK (price >= 0),
    image_url   VARCHAR(512),
    active      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_concession_items_active ON concession_items (active) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS combo_orders (
    id         UUID PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id),
    booking_id UUID        REFERENCES bookings(id),
    status     VARCHAR(16) NOT NULL DEFAULT 'confirmed',
    total      BIGINT      NOT NULL DEFAULT 0 CHECK (total >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_combo_order_status CHECK (status IN ('pending','confirmed','cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_combo_orders_user ON combo_orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_combo_orders_booking ON combo_orders (booking_id) WHERE booking_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS combo_order_items (
    id             UUID PRIMARY KEY,
    combo_order_id UUID        NOT NULL REFERENCES combo_orders(id) ON DELETE CASCADE,
    combo_id       UUID        NOT NULL REFERENCES concession_items(id),
    combo_name     VARCHAR(255) NOT NULL,
    quantity       INTEGER     NOT NULL CHECK (quantity > 0),
    unit_price     BIGINT      NOT NULL CHECK (unit_price >= 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_combo_order_items_order ON combo_order_items (combo_order_id);
