// Unwrap the embed src from an iframe snippet, drop the extra HTML, only allowlisted hosts embed.
export const extractIframeSrc = (input: string): string => {
  const match = input.match(/<iframe[^>]*\ssrc=["']([^"']+)["'][^>]*>/i);
  return match ? match[1] : input;
};

// Convert trailer_url to a nocookie embed; legacy links open the source instead of a broken iframe.
export const youtubeEmbedUrl = (trailerUrl?: string | null): string | null => {
  if (!trailerUrl) return null;

  let parsed: URL;
  try {
    parsed = new URL(extractIframeSrc(trailerUrl.trim()));
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

  // Only an 11-char id embeds.
  if (!/^[\w-]{11}$/.test(id)) return null;

  return `https://www.youtube-nocookie.com/embed/${id}?autoplay=1&rel=0`;
};
