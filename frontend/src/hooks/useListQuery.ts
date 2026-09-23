import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import type { PageQuery } from '@/types';

export const DEFAULT_PAGE_SIZE = 10;

/** Backend rejects page_size > 100 with 400/40001 instead of clamping. */
export const MAX_PAGE_SIZE = 100;

/** Param names mirror the API query so the URL reads as the request. */
const PAGE = 'page';
const PAGE_SIZE = 'page_size';
const SEARCH = 'search';

const toInt = (raw: string | null, fallback: number, min: number, max: number): number => {
  const parsed = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.min(Math.max(parsed, min), max);
};

export interface ListQueryState {
  /** Pass straight into api.list(...) and the react-query key. */
  query: PageQuery;
  page: number;
  pageSize: number;
  search: string;
  setPage: (page: number, pageSize?: number) => void;
  setSearch: (search: string) => void;
  reset: () => void;
}

/** Page/size/search live in the URL (survives F5, linkable). Defaults stay out of the URL; every change uses replace so paging doesn't pollute history. */
export const useListQuery = (defaultPageSize: number = DEFAULT_PAGE_SIZE): ListQueryState => {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = toInt(searchParams.get(PAGE), 1, 1, Number.MAX_SAFE_INTEGER);
  const pageSize = toInt(searchParams.get(PAGE_SIZE), defaultPageSize, 1, MAX_PAGE_SIZE);
  const search = searchParams.get(SEARCH)?.trim() ?? '';

  // Empty search is dropped from the query and the key alike.
  const query = useMemo<PageQuery>(
    () => ({ page, page_size: pageSize, ...(search ? { search } : {}) }),
    [page, pageSize, search]
  );

  const patch = useCallback(
    (next: Record<string, string | number | undefined>) => {
      setSearchParams(
        (current) => {
          const params = new URLSearchParams(current);
          Object.entries(next).forEach(([key, value]) => {
            if (value === undefined || value === '') params.delete(key);
            else params.set(key, String(value));
          });
          return params;
        },
        { replace: true }
      );
    },
    [setSearchParams]
  );

  const setPage = useCallback(
    (nextPage: number, nextSize?: number) => {
      const size = nextSize ?? pageSize;
      patch({
        [PAGE]: nextPage > 1 ? nextPage : undefined,
        [PAGE_SIZE]: size === defaultPageSize ? undefined : size,
      });
    },
    [patch, pageSize, defaultPageSize]
  );

  // New search resets to page 1: page 7 of the old result is almost surely empty.
  const setSearch = useCallback(
    (nextSearch: string) => {
      patch({ [SEARCH]: nextSearch.trim() || undefined, [PAGE]: undefined });
    },
    [patch]
  );

  const reset = useCallback(() => {
    patch({ [PAGE]: undefined, [PAGE_SIZE]: undefined, [SEARCH]: undefined });
  }, [patch]);

  return { query, page, pageSize, search, setPage, setSearch, reset };
};
