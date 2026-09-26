-- Nothing references backdrop_url, so the column drops on its own.
ALTER TABLE movies
    DROP COLUMN IF EXISTS backdrop_url;
