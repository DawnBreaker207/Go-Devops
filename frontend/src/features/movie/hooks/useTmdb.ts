import { useQuery } from '@tanstack/react-query';
import { useDebouncedValue } from '@/hooks/useDebouncedValue';
import { isTmdbEnabled, searchMovies } from '@/api/tmdb.api';

export const TMDB_QUERY_KEY = 'tmdb-search';

/** Debounced TMDB title search. Disabled until 2 chars or no token. */
export const useTmdbSearch = (query: string) => {
  const debounced = useDebouncedValue(query.trim(), 400);
  return useQuery({
    queryKey: [TMDB_QUERY_KEY, debounced],
    queryFn: () => searchMovies(debounced),
    enabled: isTmdbEnabled() && debounced.length >= 2,
    staleTime: 5 * 60_000,
    retry: 1,
  });
};
