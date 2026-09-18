import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import type { PageQuery } from '@/types';

export const DEFAULT_PAGE_SIZE = 10;

/** Backend tu choi page_size > 100 bang 400/40001, no khong tu cat bot gium. */
export const MAX_PAGE_SIZE = 100;

/** Ten param giong het ten query cua API, de doc URL la biet request gui gi. */
const PAGE = 'page';
const PAGE_SIZE = 'page_size';
const SEARCH = 'search';

const toInt = (raw: string | null, fallback: number, min: number, max: number): number => {
  const parsed = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.min(Math.max(parsed, min), max);
};

export interface ListQueryState {
  /** Truyen thang vao api.list(...) va vao queryKey cua react-query. */
  query: PageQuery;
  page: number;
  pageSize: number;
  search: string;
  setPage: (page: number, pageSize?: number) => void;
  setSearch: (search: string) => void;
  reset: () => void;
}

/**
 * Trang / kich thuoc trang / tu khoa nam trong URL chu khong trong useState, nen
 * F5 hay gui link cho nguoi khac deu ra dung mot man hinh.
 *
 * Gia tri mac dinh KHONG duoc ghi vao URL: /movies sach hon /movies?page=1&page_size=10
 * va hai dia chi do phai la cung mot trang. Moi thay doi dung replace: mot bang
 * admin bam qua 5 trang khong nen chen 5 muc vao lich su trinh duyet.
 */
export const useListQuery = (defaultPageSize: number = DEFAULT_PAGE_SIZE): ListQueryState => {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = toInt(searchParams.get(PAGE), 1, 1, Number.MAX_SAFE_INTEGER);
  const pageSize = toInt(searchParams.get(PAGE_SIZE), defaultPageSize, 1, MAX_PAGE_SIZE);
  const search = searchParams.get(SEARCH)?.trim() ?? '';

  // search rong thi bo han khoi query: khong gui search= thua, va queryKey cua
  // react-query cung khong doi chi vi nguoi dung xoa o tim kiem.
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

  // Doi tu khoa thi ve trang 1: trang 7 cua ket qua cu gan nhu chac chan rong.
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
