import { useCallback, useSyncExternalStore } from 'react';
import { safeStorage } from '@/utils/storage';

const STORAGE_KEY = 'cp:favorite-movies';
const listeners = new Set<() => void>();

let cache = safeStorage.get<string[]>(STORAGE_KEY) ?? [];

const write = (ids: string[]) => {
  cache = ids;
  safeStorage.set(STORAGE_KEY, ids);
  listeners.forEach((listener) => listener());
};

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => listeners.delete(listener);
};

const getSnapshot = () => cache;

/** Per-device favorite movies stored locally. */
export const useFavoriteMovies = () => {
  const ids = useSyncExternalStore(subscribe, getSnapshot);

  const isFavorite = useCallback((id: string) => ids.includes(id), [ids]);

  const toggle = useCallback(
    (id: string) => {
      write(ids.includes(id) ? ids.filter((existing) => existing !== id) : [...ids, id]);
    },
    [ids]
  );

  return { isFavorite, toggle };
};

export default useFavoriteMovies;
