-- poster_url stays NULL; real artwork is uploaded later.

INSERT INTO movies (id, title, genre, duration, director, description, release_date, status)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'The Dark Horizon',   'Sci-Fi',   128, 'Ava Lane',   'A crew routes humanity''s last colony ship through an uncharted nebula.', '2026-01-15', 'showing'),
    ('10000000-0000-0000-0000-000000000002', 'Golden Drift',       'Drama',    112, 'Tian Lu',    'Two estranged siblings reopen their late father''s highland tea farm.', '2026-03-05', 'showing'),
    ('10000000-0000-0000-0000-000000000003', 'City of Echoes',     'Mystery',  105, 'Dan Okafor', 'A sound engineer uncovers a crime hidden in archival recordings.',      '2026-04-10', 'showing'),
    ('10000000-0000-0000-0000-000000000004', 'Last Ticket Home',   'Romance',  98,  'Yuki Mori',  'A cancelled flight strands two strangers in the same terminal.',        '2026-05-20', 'showing'),
    ('10000000-0000-0000-0000-000000000005', 'Code: Midnight',     'Action',   121, 'R. Vasquez', 'A night-shift coder becomes the only witness to a digital heist.',      '2026-06-30', 'showing')
ON CONFLICT (id) DO NOTHING;