import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { movieApi } from '@/api/movie.api';
import { BROWSE_QUERY_KEY } from '@/features/browse/hooks/useBrowse';
import type { MoviePayload, MovieStatus, PageQuery } from '@/types';

export const MOVIE_QUERY_KEY = 'movies';

/** Refresh all movie caches after a mutation. */
const invalidateMovieCaches = (queryClient: ReturnType<typeof useQueryClient>) => {
  queryClient.invalidateQueries({ queryKey: [MOVIE_QUERY_KEY] });
  queryClient.invalidateQueries({ queryKey: [BROWSE_QUERY_KEY] });
};

export const useMovieList = (query: PageQuery & { status?: MovieStatus }) =>
  useQuery({
    queryKey: [MOVIE_QUERY_KEY, query],
    queryFn: () => movieApi.list(query),
    placeholderData: (previous) => previous,
    // The same admin is editing this list (proactive invalidate above) -
    // no refetch-on-mount needed within one short working session.
    staleTime: 30_000,
  });

export const useCreateMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: MoviePayload) => movieApi.create(payload),
    onSuccess: () => invalidateMovieCaches(queryClient),
  });
};

export const useUpdateMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: MoviePayload }) =>
      movieApi.update(id, payload),
    onSuccess: () => invalidateMovieCaches(queryClient),
  });
};

export const useDeleteMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => movieApi.remove(id),
    onSuccess: () => invalidateMovieCaches(queryClient),
  });
};
