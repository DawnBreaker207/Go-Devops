-- Module combo inventory per branch. No row for a (branch, combo) pair means
-- unlimited stock. Reserved at HOLD time, released on cancel/expire/refund.
CREATE TABLE IF NOT EXISTS combo_branch_stock (
    branch_id      UUID   NOT NULL REFERENCES branches(id),
    combo_id       UUID   NOT NULL REFERENCES combos(id),
    stock_quantity INTEGER NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (branch_id, combo_id),
    CONSTRAINT ck_combo_branch_stock_qty CHECK (stock_quantity >= 0)
);
