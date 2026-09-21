-- Seed data: sample movies so a fresh DB has content for the UI.
-- NOT tracked by golang-migrate (schema_migrations lives in migrations/schema);
-- apply with `make migrate-seed` (runs every *.sql here via psql in order).
-- Idempotent: fixed IDs + ON CONFLICT DO NOTHING, re-running changes nothing.
-- poster_url: TAM THOI dung anh placeholder (picsum.photos, co dinh theo seed
-- tung phim) de UI co gi de xem trong luc dev - thay bang anh that/upload
-- qua Cloudinary khi co artwork chinh thuc.

INSERT INTO movies (id, title, genre, duration, director, description, release_date, status, poster_url)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'The Dark Horizon',   'Sci-Fi',   128, 'Ava Lane',   'A crew routes humanity''s last colony ship through an uncharted nebula.', '2026-01-15', 'showing', 'https://picsum.photos/seed/dark-horizon/500/750'),
    ('10000000-0000-0000-0000-000000000002', 'Golden Drift',       'Drama',    112, 'Tian Lu',    'Two estranged siblings reopen their late father''s highland tea farm.', '2026-03-05', 'showing', 'https://picsum.photos/seed/golden-drift/500/750'),
    ('10000000-0000-0000-0000-000000000003', 'City of Echoes',     'Mystery',  105, 'Dan Okafor', 'A sound engineer uncovers a crime hidden in archival recordings.',      '2026-04-10', 'showing', 'https://picsum.photos/seed/city-of-echoes/500/750'),
    ('10000000-0000-0000-0000-000000000004', 'Last Ticket Home',   'Romance',  98,  'Yuki Mori',  'A cancelled flight strands two strangers in the same terminal.',        '2026-05-20', 'showing', 'https://picsum.photos/seed/last-ticket-home/500/750'),
    ('10000000-0000-0000-0000-000000000005', 'Code: Midnight',     'Action',   121, 'R. Vasquez', 'A night-shift coder becomes the only witness to a digital heist.',      '2026-06-30', 'showing', 'https://picsum.photos/seed/code-midnight/500/750'),
    -- coming_soon: de tab "Sap chieu" cua UI co du lieu de xem thay vi rong.
    ('10000000-0000-0000-0000-000000000006', 'Silent Frequency',  'Thriller', 115, 'Nadia Cross', 'A radio host starts receiving broadcasts from a station that went dark decades ago.', '2026-11-01', 'coming_soon', 'https://picsum.photos/seed/silent-frequency/500/750'),
    ('10000000-0000-0000-0000-000000000007', 'Paper Lanterns',    'Family',   100, 'Minh Tran',  'A grandmother teaches her grandchildren the lantern-making craft of a fading village.', '2026-12-20', 'coming_soon', 'https://picsum.photos/seed/paper-lanterns/500/750')
ON CONFLICT (id) DO NOTHING;