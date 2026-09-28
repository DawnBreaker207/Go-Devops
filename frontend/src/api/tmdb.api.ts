import axios from 'axios';

const TMDB_API = 'https://api.themoviedb.org/3';
const TMDB_IMAGE = 'https://image.tmdb.org/t/p';
const LANG = 'vi-VN';

const token = (): string => import.meta.env.VITE_TMDB_READ_TOKEN ?? '';

export const isTmdbEnabled = (): boolean => token().length > 0;

const tmdb = axios.create({ baseURL: TMDB_API, timeout: 15_000 });

const authHeader = () => ({ Authorization: `Bearer ${token()}` });

export interface TmdbSearchResult {
  id: number;
  title: string;
  release_date?: string;
  overview?: string;
  poster_thumb?: string;
}

interface TmdbSearchResponse {
  results?: Array<{
    id: number;
    title: string;
    release_date?: string;
    overview?: string;
    poster_path?: string | null;
  }>;
}

export const searchMovies = async (query: string): Promise<TmdbSearchResult[]> => {
  const { data } = await tmdb.get<TmdbSearchResponse>('/search/movie', {
    headers: authHeader(),
    params: { query, language: LANG, page: 1 },
  });
  return (data.results ?? []).map((r) => ({
    id: r.id,
    title: r.title,
    release_date: r.release_date,
    overview: r.overview,
    poster_thumb: r.poster_path ? `${TMDB_IMAGE}/w342${r.poster_path}` : undefined,
  }));
};

export interface TmdbMovieDraft {
  title: string;
  genre?: string;
  duration?: number;
  director?: string;
  cast?: string;
  release_date?: string;
  description?: string;
  poster_url?: string;
  backdrop_url?: string;
  trailer_url?: string;
}

interface TmdbDetailsResponse {
  title: string;
  runtime?: number | null;
  release_date?: string;
  overview?: string;
  poster_path?: string | null;
  backdrop_path?: string | null;
  genres?: Array<{ name: string }>;
  credits?: {
    cast?: Array<{ name: string }>;
    crew?: Array<{ name: string; job: string }>;
  };
  videos?: {
    results?: Array<{ site: string; type: string; key: string }>;
  };
}

export const fetchMovieDraft = async (id: number): Promise<TmdbMovieDraft> => {
  const params = { language: LANG, append_to_response: 'credits,videos' };
  const { data } = await tmdb.get<TmdbDetailsResponse>(`/movie/${id}`, {
    headers: authHeader(),
    params,
  });
  let videos = data.videos?.results ?? [];
  if (!videos.some((v) => v.site === 'YouTube' && v.type === 'Trailer')) {
    const fallback = await tmdb.get<TmdbDetailsResponse>(`/movie/${id}`, {
      headers: authHeader(),
      params: { ...params, language: 'en-US' },
    });
    videos = fallback.data.videos?.results ?? [];
  }
  const director = data.credits?.crew?.find((c) => c.job === 'Director')?.name;
  const cast = (data.credits?.cast ?? []).slice(0, 5).map((c) => c.name);
  const trailer = videos.find((v) => v.site === 'YouTube' && v.type === 'Trailer');
  return {
    title: data.title,
    genre: data.genres?.[0]?.name.replace(/^Phim\s/i, ''),
    duration: data.runtime ?? undefined,
    director,
    cast: cast.length > 0 ? cast.join(', ') : undefined,
    release_date: data.release_date || undefined,
    description: data.overview || undefined,
    poster_url: data.poster_path ? `${TMDB_IMAGE}/original${data.poster_path}` : undefined,
    backdrop_url: data.backdrop_path ? `${TMDB_IMAGE}/original${data.backdrop_path}` : undefined,
    trailer_url: trailer ? `https://www.youtube.com/watch?v=${trailer.key}` : undefined,
  };
};
