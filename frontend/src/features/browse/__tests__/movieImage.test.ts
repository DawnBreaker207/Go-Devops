import { describe, expect, it } from 'vitest';
import { movieBackdrop, movieYear } from '../movieImage';

describe('movieBackdrop', () => {
  it('prefers the landscape backdrop', () => {
    expect(
      movieBackdrop({ backdrop_url: 'https://cdn/bd.jpg', poster_url: 'https://cdn/p.jpg' })
    ).toBe('https://cdn/bd.jpg');
  });

  it('falls back to the poster', () => {
    // Every movie created before migration 000011 has an empty backdrop; the home must still render.
    expect(movieBackdrop({ backdrop_url: '', poster_url: 'https://cdn/p.jpg' })).toBe(
      'https://cdn/p.jpg'
    );
  });

  it('returns an empty string when the movie has no image at all', () => {
    // The card draws its initial-letter placeholder on '', so this must not become undefined.
    expect(movieBackdrop({ backdrop_url: '', poster_url: '' })).toBe('');
  });
});

describe('movieYear', () => {
  it('reads the year off the YYYY-MM-DD release date', () => {
    expect(movieYear({ release_date: '2026-09-25' })).toBe('2026');
  });

  it('does not shift the year across a timezone boundary', () => {
    // The reason this slices instead of parsing: 1 Jan in +07:00 is 31 Dec elsewhere, and a
    // date-only string run through a zone would report the wrong year.
    expect(movieYear({ release_date: '2026-01-01' })).toBe('2026');
    expect(movieYear({ release_date: '2026-12-31' })).toBe('2026');
  });

  it('returns empty for anything that is not a plain date', () => {
    expect(movieYear({ release_date: '' })).toBe('');
    expect(movieYear({ release_date: 'not-a-date' })).toBe('');
  });
});
