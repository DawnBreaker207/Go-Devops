-- Seed data: the concession catalogue (popcorn/drinks) behind step 2 of the
-- booking wizard. NOT tracked by golang-migrate; apply with `make migrate-seed`.
-- Idempotent: fixed IDs + ON CONFLICT DO NOTHING.
--
-- Why this file exists: `concession_items` shipped empty with no seed and no
-- admin write path, so "Chon combo" was permanently blank on every fresh
-- environment and the only way to fill it was hand-written SQL. The write path
-- now exists (/api/v1/admin/concessions), but a fresh DB still needs something
-- on the shelves for the flow to be demonstrable.
--
-- Prices are int64 whole VND like every other amount in this API — no minor
-- units, no currency column. `image_url` is left NULL on purpose: the repo
-- hosts no concession artwork, and a broken <img> reads worse than none.

INSERT INTO concession_items (id, name, description, price, active)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'Combo 1 - Bap ngot + 1 Coca',
     'Mot bap rang bo vi ngot (60oz) va mot Coca-Cola 32oz.', 89000, TRUE),
    ('30000000-0000-0000-0000-000000000002', 'Combo 2 - Bap ngot + 2 Coca',
     'Mot bap rang bo vi ngot (60oz) va hai Coca-Cola 32oz. Vua doi.', 119000, TRUE),
    ('30000000-0000-0000-0000-000000000003', 'Combo Gia dinh',
     'Hai bap rang bo (60oz), bon nuoc ngot 32oz va mot khoai tay chien.', 219000, TRUE),
    ('30000000-0000-0000-0000-000000000004', 'Bap rang bo (60oz)',
     'Mot bap rang bo co lon, chon vi ngot hoac vi pho mai tai quay.', 59000, TRUE),
    ('30000000-0000-0000-0000-000000000005', 'Coca-Cola 32oz',
     'Mot ly Coca-Cola co lon kem da.', 35000, TRUE),
    ('30000000-0000-0000-0000-000000000006', 'Nuoc suoi Dasani 500ml',
     'Mot chai nuoc suoi 500ml.', 20000, TRUE),
    -- One deliberately inactive row: it must NOT appear in GET /combos (public)
    -- but MUST appear in GET /admin/concessions, which is the whole difference
    -- between the two list endpoints. Keeping it here means the distinction is
    -- exercised on any fresh environment instead of only in tests.
    ('30000000-0000-0000-0000-000000000007', 'Combo Tet (ngung ban)',
     'Combo theo mua, da het thoi gian ban. Giu lai de thu lai khi can.', 149000, FALSE)
ON CONFLICT (id) DO NOTHING;
