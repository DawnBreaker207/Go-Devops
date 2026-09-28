import { describe, expect, it } from 'vitest';
import { extractIframeSrc, youtubeEmbedUrl } from '../trailerEmbed';

const ID = 'YoHD9XEInc0';

describe('extractIframeSrc', () => {
  it('pulls the src out of a pasted <iframe> embed snippet', () => {
    const snippet = `<iframe width="560" height="315" src="https://www.youtube.com/embed/${ID}" title="YouTube video player" frameborder="0" allowfullscreen></iframe>`;
    expect(extractIframeSrc(snippet)).toBe(`https://www.youtube.com/embed/${ID}`);
  });

  it('returns the input unchanged when it is not an iframe snippet', () => {
    expect(extractIframeSrc(`https://youtu.be/${ID}`)).toBe(`https://youtu.be/${ID}`);
  });
});

describe('youtubeEmbedUrl', () => {
  it('embeds a pasted <iframe> snippet, not just a bare link', () => {
    const snippet = `<iframe src="https://www.youtube.com/embed/${ID}"></iframe>`;
    expect(youtubeEmbedUrl(snippet)).toBe(
      `https://www.youtube-nocookie.com/embed/${ID}?autoplay=1&rel=0`
    );
  });

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
