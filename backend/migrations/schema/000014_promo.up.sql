-- Module promo: discount codes, holiday campaigns grouping them with combos
-- and articles, and the per-account redemption guard (PLAN_CAMPAIGN.md).
--
-- WHY `bookings.total_amount` IS NOT DISCOUNTED
-- At confirm time the service asserts that the sum of the sold seats' prices
-- equals total_amount (see finalizeTx in internal/service/booking_payment.go);
-- a booking that fails that assertion cannot confirm AFTER the money has
-- already been taken. So total_amount stays the seat subtotal, discount_amount
-- is stored beside it on bookings (000003), and the amount actually charged
-- is (total_amount - discount_amount), which is what goes into
-- payments.amount. Every downstream money path (amount-mismatch detection,
-- refunds) already reads payments.amount, so they follow automatically.

CREATE TABLE IF NOT EXISTS campaigns (
    id             UUID         PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    starts_at      TIMESTAMPTZ  NOT NULL,
    ends_at        TIMESTAMPTZ  NOT NULL,
    active         BOOLEAN      NOT NULL,
    -- v1 only ever enforces 1 (discount_redemptions' UNIQUE below is a hard
    -- (user, code) pair, not a count against this column).
    per_user_limit INTEGER      NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_campaigns_window CHECK (ends_at > starts_at),
    CONSTRAINT ck_campaigns_per_user_limit CHECK (per_user_limit >= 1)
);

CREATE INDEX IF NOT EXISTS idx_campaigns_active_window ON campaigns (active, starts_at, ends_at);

-- Alter discount_codes (created by 000010) instead of recreating it:
ALTER TABLE IF EXISTS discount_codes ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_discount_codes_campaign
    ON discount_codes (campaign_id) WHERE campaign_id IS NOT NULL;

-- v1 is record/display only: promo_price is shown next to the campaign but
-- does NOT override concession_items.price - the actual sell price is still
-- PATCHed by hand via /admin/concessions/:id.
CREATE TABLE IF NOT EXISTS campaign_combos (
    campaign_id UUID    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    combo_id    UUID    NOT NULL REFERENCES concession_items(id),
    promo_price INTEGER NULL,

    PRIMARY KEY (campaign_id, combo_id),
    CONSTRAINT ck_campaign_combos_promo_price CHECK (promo_price IS NULL OR promo_price >= 0)
);

CREATE TABLE IF NOT EXISTS campaign_articles (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    article_id  UUID NOT NULL REFERENCES articles(id),

    PRIMARY KEY (campaign_id, article_id)
);

-- The per-account guard. Deliberately keyed per (user, discount_code_id), NOT
-- per (user, campaign_id): a customer may redeem several DIFFERENT codes
-- belonging to the same campaign - only re-using the SAME code a second time
-- is blocked. ClaimRedemption's unique-violation IS the concurrency control
-- (no prior SELECT, so two simultaneous applies by the same user can never
-- both win). Discount codes are soft-deleted, never removed, so the
-- reference stays valid.
CREATE TABLE IF NOT EXISTS discount_redemptions (
    id               UUID        PRIMARY KEY,
    user_id          UUID        NOT NULL REFERENCES users(id),
    discount_code_id UUID        NOT NULL REFERENCES discount_codes(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_discount_redemptions_user_code UNIQUE (user_id, discount_code_id)
);

CREATE INDEX IF NOT EXISTS idx_discount_redemptions_code ON discount_redemptions (discount_code_id);
