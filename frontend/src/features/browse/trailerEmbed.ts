/** Turn a stored `trailer_url` into a privacy-friendly YouTube embed URL.
 *
 *  The dev data uses `https://www.youtube.com/watch?v=<id>`, but operators paste whatever they have,
 *  so `youtu.be/<id>` and an already-embedded `/embed/<id>` are accepted too. Anything else returns
 *  null and the caller falls back to opening the raw link in a new tab rather than rendering a
 *  broken iframe.
 *
 *  `youtube-nocookie.com` is deliberate: nothing is requested from YouTube until the viewer actually
 *  clicks play, and when they do it is the no-cookie host. */
export const youtubeEmbedUrl = (trailerUrl?: string | null): string | null => {
  if (!trailerUrl) return null;

  let parsed: URL;
  try {
    parsed = new URL(trailerUrl);
  } catch {
    return null;
  }

  const host = parsed.hostname.replace(/^www\./, '');
  let id = '';

  if (host === 'youtu.be') {
    id = parsed.pathname.slice(1);
  } else if (
    host === 'youtube.com' ||
    host === 'm.youtube.com' ||
    host === 'youtube-nocookie.com'
  ) {
    id = parsed.searchParams.get('v') ?? '';
    if (!id && parsed.pathname.startsWith('/embed/')) id = parsed.pathname.slice('/embed/'.length);
  }

  // Ids are 11 chars of [A-Za-z0-9_-]; anything else is not something we can embed.
  if (!/^[\w-]{11}$/.test(id)) return null;

  return `https://www.youtube-nocookie.com/embed/${id}?autoplay=1&rel=0`;
};
