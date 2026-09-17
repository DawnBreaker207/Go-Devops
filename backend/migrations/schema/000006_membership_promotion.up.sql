-- Module membership & promotion: voucher, membership tier, combo, article,
-- admin permission groups, and the owner role.

ALTER TABLE users DROP CONSTRAINT ck_user_role;
ALTER TABLE users ADD CONSTRAINT ck_user_role CHECK (role IN ('customer','staff','admin','owner'));

CREATE TABLE IF NOT EXISTS vouchers (
    id                  UUID PRIMARY KEY,
    code                VARCHAR(32)  NOT NULL,
    discount_type       VARCHAR(16)  NOT NULL,
    discount_value      BIGINT       NOT NULL,
    max_discount        BIGINT,
    min_order_amount    BIGINT       NOT NULL DEFAULT 0,
    starts_at           TIMESTAMPTZ  NOT NULL,
    ends_at             TIMESTAMPTZ  NOT NULL,
    max_usage           INTEGER      NOT NULL,
    usage_count         INTEGER      NOT NULL DEFAULT 0,
    max_usage_per_user  INTEGER      NOT NULL DEFAULT 1,
    apply_scope         VARCHAR(16)  NOT NULL DEFAULT 'all',
    branch_id           UUID,
    campaign_id         UUID,
    created_by_admin_id UUID         NOT NULL REFERENCES users(id),
    status              VARCHAR(20)  NOT NULL DEFAULT 'pending_approval',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_voucher_discount_type CHECK (discount_type IN ('fixed','percentage')),
    CONSTRAINT ck_voucher_apply_scope CHECK (apply_scope IN ('seat_only','combo_only','all')),
    CONSTRAINT ck_voucher_status CHECK (status IN ('pending_approval','active','rejected','disabled')),
    CONSTRAINT ck_voucher_discount_value CHECK (discount_value > 0),
    CONSTRAINT ck_voucher_max_discount CHECK (max_discount IS NULL OR max_discount > 0),
    CONSTRAINT ck_voucher_min_order CHECK (min_order_amount >= 0),
    CONSTRAINT ck_voucher_dates CHECK (ends_at > starts_at),
    CONSTRAINT ck_voucher_max_usage CHECK (max_usage > 0),
    CONSTRAINT ck_voucher_max_usage_per_user CHECK (max_usage_per_user > 0),
    CONSTRAINT ck_voucher_usage_count CHECK (usage_count >= 0 AND usage_count <= max_usage)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_code ON vouchers (code);
CREATE INDEX IF NOT EXISTS idx_vouchers_status_dates ON vouchers (status, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS voucher_redemptions (
    id         UUID PRIMARY KEY,
    voucher_id UUID        NOT NULL REFERENCES vouchers(id),
    user_id    UUID        NOT NULL REFERENCES users(id),
    booking_id UUID        NOT NULL REFERENCES bookings(id),
    used_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_voucher_redemptions_booking UNIQUE (booking_id)
);

CREATE INDEX IF NOT EXISTS idx_voucher_redemptions_voucher_user ON voucher_redemptions (voucher_id, user_id);

CREATE TABLE IF NOT EXISTS membership_tiers (
    id                  UUID          PRIMARY KEY,
    name                VARCHAR(128)  NOT NULL,
    price               BIGINT        NOT NULL,
    duration_days       INTEGER       NOT NULL,
    discount_percent    NUMERIC(5,2)  NOT NULL,
    excluded_seat_types JSONB,
    active              BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_membership_tier_price CHECK (price > 0),
    CONSTRAINT ck_membership_tier_duration CHECK (duration_days > 0),
    CONSTRAINT ck_membership_tier_discount CHECK (discount_percent > 0 AND discount_percent <= 100)
);

CREATE TABLE IF NOT EXISTS user_memberships (
    id              UUID         PRIMARY KEY,
    user_id         UUID         NOT NULL REFERENCES users(id),
    tier_id         UUID         NOT NULL REFERENCES membership_tiers(id),
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ  NOT NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    idempotency_key VARCHAR(128),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_user_membership_status CHECK (status IN ('active','expired'))
);

CREATE UNIQUE INDEX IF NOT EXISTS one_active_membership_per_user
    ON user_memberships (user_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_user_memberships_user ON user_memberships (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_memberships_idem
    ON user_memberships (idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS combos (
    id           UUID         PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    price        BIGINT       NOT NULL,
    member_price BIGINT,
    active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_combo_price CHECK (price > 0),
    CONSTRAINT ck_combo_member_price CHECK (member_price IS NULL OR member_price > 0)
);

CREATE TABLE IF NOT EXISTS booking_combos (
    id           UUID         PRIMARY KEY,
    booking_id   UUID         NOT NULL REFERENCES bookings(id),
    combo_id     UUID         NOT NULL REFERENCES combos(id),
    quantity     INTEGER      NOT NULL,
    price        BIGINT       NOT NULL,
    delivered    BOOLEAN      NOT NULL DEFAULT FALSE,
    delivered_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_booking_combo_quantity CHECK (quantity > 0),
    CONSTRAINT ck_booking_combo_price CHECK (price > 0),
    CONSTRAINT uq_booking_combos_booking_combo UNIQUE (booking_id, combo_id)
);

CREATE INDEX IF NOT EXISTS idx_booking_combos_booking ON booking_combos (booking_id);

CREATE TABLE IF NOT EXISTS articles (
    id            UUID         PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL,
    summary       VARCHAR(500),
    thumbnail_url VARCHAR(1024),
    content       TEXT         NOT NULL,
    author_id     UUID         NOT NULL REFERENCES users(id),
    type          VARCHAR(16)  NOT NULL DEFAULT 'news',
    status        VARCHAR(16)  NOT NULL DEFAULT 'draft',
    views         BIGINT       NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT ck_article_type CHECK (type IN ('news','promotion')),
    CONSTRAINT ck_article_status CHECK (status IN ('draft','published','hidden'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_articles_slug ON articles (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_articles_status_created ON articles (status, created_at);

CREATE TABLE IF NOT EXISTS admin_permissions (
    id             UUID        PRIMARY KEY,
    user_id        UUID        NOT NULL REFERENCES users(id),
    permission_key VARCHAR(16) NOT NULL,
    granted_by     UUID        NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_admin_permission_key CHECK (permission_key IN ('content','pricing','finance','accounts')),
    CONSTRAINT uq_admin_permissions_user_key UNIQUE (user_id, permission_key)
);

CREATE INDEX IF NOT EXISTS idx_admin_permissions_user ON admin_permissions (user_id);

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS voucher_id UUID REFERENCES vouchers(id);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS voucher_discount_amount BIGINT NOT NULL DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS membership_id UUID REFERENCES user_memberships(id);

-- audit_logs is append-only even for the table owner; only the retention
-- cleanup job (internal/jobs/cleanup.go) may set this flag to delete.
CREATE OR REPLACE FUNCTION audit_logs_immutable() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' AND current_setting('audit.allow_retention_delete', true) = 'on' THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'audit_logs is append-only: % is not allowed', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
CREATE TRIGGER trg_audit_logs_immutable
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_immutable();
