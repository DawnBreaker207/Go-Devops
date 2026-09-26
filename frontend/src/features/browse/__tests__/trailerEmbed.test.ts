import { describe, expect, it } from 'vitest';
import { youtubeEmbedUrl } from '../trailerEmbed';

const ID = 'YoHD9XEInc0';

describe('youtubeEmbedUrl', () => {
  it('embeds the watch URL the dev data actually stores', () => {
    expect(youtubeEmbedUrl(`https://www.youtube.com/watch?v=${ID}`)).toBe(
      `https://www.youtube-nocookie.com/embed/${ID}?autoplay=1&rel=0`
    );
  });

  it('accepts the short link and an already-embedded link', () => {
    // Operators paste whatever their browser gave them.
    expect(youtubeEmbedUrl(`https://youtu.be/${ID}`)).toContain(`/embed/${ID}`);
    expect(youtubeEmbedUrl(`https://www.youtube.com/embed/${ID}`)).toContain(`/embed/${ID}`);
    expect(youtubeEmbedUrl(`https://m.youtube.com/watch?v=${ID}&t=42`)).toContain(`/embed/${ID}`);
  });

  it('uses the no-cookie host', () => {
    expect(youtubeEmbedUrl(`https://www.youtube.com/watch?v=${ID}`)).toContain(
      'youtube-nocookie.com'
    );
  });

  it('returns null for anything it cannot embed, so the caller opens a tab instead', () => {
    expect(youtubeEmbedUrl('')).toBeNull();
    expect(youtubeEmbedUrl(null)).toBeNull();
    expect(youtubeEmbedUrl('not a url')).toBeNull();
    expect(youtubeEmbedUrl('https://vimeo.com/12345')).toBeNull();
    expect(youtubeEmbedUrl('https://www.youtube.com/')).toBeNull();
    // An 11-character id is the whole format; a short or long one is not a video.
    expect(youtubeEmbedUrl('https://youtu.be/tooshort')).toBeNull();
  });
});
