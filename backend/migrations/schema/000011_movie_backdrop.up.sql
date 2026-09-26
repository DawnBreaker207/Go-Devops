-- A LANDSCAPE still for each movie, used by the customer home: the hero banner
-- and the "Now showing" cards are wide (16:9), while `poster_url` holds a 2:3
-- portrait. Reusing the poster there crops the subject out of frame, so the two
-- images are separate columns rather than one URL stretched two ways.
--
-- WHY IT IS NULLABLE WITH NO DEFAULT
-- Deliberate. A `default:` tag on the Go field would make GORM omit the field
-- from an INSERT whenever it is the zero value (the same trap that shipped a
-- combo as ON SALE after being created with active:false), so a movie created
-- without a backdrop would silently come back carrying one. The empty string is
-- a legitimate value here and the frontend falls back to poster_url, so the
-- column carries no DEFAULT and no NOT NULL.

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS backdrop_url VARCHAR(512);
