import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { movieApi } from '@/api/movie.api';
import type { MoviePayload, PageQuery } from '@/types';

export const MOVIE_QUERY_KEY = 'movies';

export const useMovieList = (query: PageQuery) =>
  useQuery({
    queryKey: [MOVIE_QUERY_KEY, query],
    queryFn: () => movieApi.list(query),
    placeholderData: (previous) => previous,
  });

export const useCreateMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: MoviePayload) => movieApi.create(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [MOVIE_QUERY_KEY] }),
  });
};

export const useUpdateMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: MoviePayload }) =>
      movieApi.update(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [MOVIE_QUERY_KEY] }),
  });
};

export const useDeleteMovie = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => movieApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [MOVIE_QUERY_KEY] }),
  });
};
