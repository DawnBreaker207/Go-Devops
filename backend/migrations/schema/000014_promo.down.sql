DROP TABLE IF EXISTS discount_redemptions;
DROP TABLE IF EXISTS campaign_articles;
DROP TABLE IF EXISTS campaign_combos;
ALTER TABLE IF EXISTS discount_codes DROP COLUMN IF EXISTS campaign_id;
DROP TABLE IF EXISTS campaigns;
