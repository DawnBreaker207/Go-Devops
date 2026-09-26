-- Seed data: sample movies so a fresh DB has content for the UI.
-- NOT tracked by golang-migrate (schema_migrations lives in migrations/schema);
-- apply with `make migrate-seed` (runs every *.sql here via psql in order).
-- Idempotent: fixed IDs + ON CONFLICT DO NOTHING, re-running changes nothing.
-- poster_url: TAM THOI dung anh placeholder (picsum.photos, co dinh theo seed
-- tung phim) de UI co gi de xem trong luc dev - thay bang anh that/upload
-- qua Cloudinary khi co artwork chinh thuc.
-- backdrop_url: anh NGANG (1280x720) cho trang chu khach - hero banner va the
-- phim deu 3:2, poster 2:3 cat mat chu the. Seed rieng ("-bd") de anh nen KHAC
-- anh poster, neu khong thi khong phan biet duoc hai cot khi nhin bang mat.
-- trailer_url: MOT video cong khai dung chung cho ca 7 phim - cung tinh chat tam
-- thoi nhu picsum. Thieu no thi khoi "Trailer" o trang chu tu an di, nen day la
-- du lieu bat buoc de trang chu hien dung nhu tren may dev.
--
-- `ON CONFLICT (id) DO NOTHING` khong backfill duoc mot DB da co san cac phim nay,
-- nen phia duoi co them buoc UPDATE chi dien vao cac o CON TRONG. No khong bao gio
-- ghi de anh that ma nguoi van hanh da upload.

INSERT INTO movies (id, title, genre, duration, director, description, release_date, status, poster_url, backdrop_url, trailer_url)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'The Dark Horizon',   'Sci-Fi',   128, 'Ava Lane',   'A crew routes humanity''s last colony ship through an uncharted nebula.', '2026-01-15', 'showing', 'https://picsum.photos/seed/dark-horizon/500/750', 'https://picsum.photos/seed/dark-horizon-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    ('10000000-0000-0000-0000-000000000002', 'Golden Drift',       'Drama',    112, 'Tian Lu',    'Two estranged siblings reopen their late father''s highland tea farm.', '2026-03-05', 'showing', 'https://picsum.photos/seed/golden-drift/500/750', 'https://picsum.photos/seed/golden-drift-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    ('10000000-0000-0000-0000-000000000003', 'City of Echoes',     'Mystery',  105, 'Dan Okafor', 'A sound engineer uncovers a crime hidden in archival recordings.',      '2026-04-10', 'showing', 'https://picsum.photos/seed/city-of-echoes/500/750', 'https://picsum.photos/seed/city-of-echoes-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    ('10000000-0000-0000-0000-000000000004', 'Last Ticket Home',   'Romance',  98,  'Yuki Mori',  'A cancelled flight strands two strangers in the same terminal.',        '2026-05-20', 'showing', 'https://picsum.photos/seed/last-ticket-home/500/750', 'https://picsum.photos/seed/last-ticket-home-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    ('10000000-0000-0000-0000-000000000005', 'Code: Midnight',     'Action',   121, 'R. Vasquez', 'A night-shift coder becomes the only witness to a digital heist.',      '2026-06-30', 'showing', 'https://picsum.photos/seed/code-midnight/500/750', 'https://picsum.photos/seed/code-midnight-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    -- coming_soon: de tab "Sap chieu" cua UI co du lieu de xem thay vi rong.
    ('10000000-0000-0000-0000-000000000006', 'Silent Frequency',  'Thriller', 115, 'Nadia Cross', 'A radio host starts receiving broadcasts from a station that went dark decades ago.', '2026-11-01', 'coming_soon', 'https://picsum.photos/seed/silent-frequency/500/750', 'https://picsum.photos/seed/silent-frequency-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0'),
    ('10000000-0000-0000-0000-000000000007', 'Paper Lanterns',    'Family',   100, 'Minh Tran',  'A grandmother teaches her grandchildren the lantern-making craft of a fading village.', '2026-12-20', 'coming_soon', 'https://picsum.photos/seed/paper-lanterns/500/750', 'https://picsum.photos/seed/paper-lanterns-bd/1280/720', 'https://www.youtube.com/watch?v=YoHD9XEInc0')
ON CONFLICT (id) DO NOTHING;

-- Backfill cho DB cu: chi dien khi o dang TRONG, khong dong vao du lieu that.
UPDATE movies m
SET backdrop_url = v.backdrop_url
FROM (VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid, 'https://picsum.photos/seed/dark-horizon-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000002'::uuid, 'https://picsum.photos/seed/golden-drift-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000003'::uuid, 'https://picsum.photos/seed/city-of-echoes-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000004'::uuid, 'https://picsum.photos/seed/last-ticket-home-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000005'::uuid, 'https://picsum.photos/seed/code-midnight-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000006'::uuid, 'https://picsum.photos/seed/silent-frequency-bd/1280/720'),
    ('10000000-0000-0000-0000-000000000007'::uuid, 'https://picsum.photos/seed/paper-lanterns-bd/1280/720')
) AS v(id, backdrop_url)
WHERE m.id = v.id AND (m.backdrop_url IS NULL OR m.backdrop_url = '');

UPDATE movies
SET trailer_url = 'https://www.youtube.com/watch?v=YoHD9XEInc0'
WHERE id::text LIKE '10000000-0000-0000-0000-%'
  AND (trailer_url IS NULL OR trailer_url = '');
