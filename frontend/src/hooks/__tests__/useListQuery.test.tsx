import { useEffect, type ReactNode } from 'react';
import { describe, expect, it } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { useListQuery, MAX_PAGE_SIZE } from '../useListQuery';

let currentSearch = '';

// Write to the outer variable in the effect, never during render: renders must stay pure.
const LocationProbe = () => {
  const { search } = useLocation();
  useEffect(() => {
    currentSearch = search;
  }, [search]);
  return null;
};

const renderListQuery = (route = '/movies', defaultPageSize?: number) => {
  currentSearch = '';
  const wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[route]}>
      {children}
      <LocationProbe />
    </MemoryRouter>
  );
  return renderHook(() => useListQuery(defaultPageSize), { wrapper });
};

describe('useListQuery', () => {
  it('mac dinh la trang 1 va khong ghi gi vao URL', () => {
    const { result } = renderListQuery();

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(10);
    expect(result.current.search).toBe('');
    expect(result.current.query).toEqual({ page: 1, page_size: 10 });
    expect(currentSearch).toBe('');
  });

  it('doc lai duoc trang tu URL, nen F5 va gui link ra cung mot man hinh', () => {
    const { result } = renderListQuery('/movies?page=3&page_size=25&search=nolan');

    expect(result.current.query).toEqual({ page: 3, page_size: 25, search: 'nolan' });
  });

  it('bo qua gia tri rac va cat page_size theo tran cua backend', () => {
    const { result } = renderListQuery('/movies?page=abc&page_size=500');

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(MAX_PAGE_SIZE);
  });

  it('setPage chi ghi phan khac mac dinh', () => {
    const { result } = renderListQuery();

    act(() => result.current.setPage(2));
    expect(currentSearch).toBe('?page=2');
    expect(result.current.query.page).toBe(2);

    act(() => result.current.setPage(1, 50));
    expect(currentSearch).toBe('?page_size=50');
    expect(result.current.query).toEqual({ page: 1, page_size: 50 });
  });

  it('doi tu khoa thi ve trang 1, va tu khoa rong bien khoi query', () => {
    const { result } = renderListQuery('/movies?page=4');

    act(() => result.current.setSearch('  dune  '));
    expect(result.current.page).toBe(1);
    expect(result.current.query).toEqual({ page: 1, page_size: 10, search: 'dune' });

    act(() => result.current.setSearch(''));
    expect(result.current.query).toEqual({ page: 1, page_size: 10 });
    expect(currentSearch).toBe('');
  });

  it('reset xoa sach moi param', () => {
    const { result } = renderListQuery('/movies?page=3&page_size=50&search=x');

    act(() => result.current.reset());
    expect(currentSearch).toBe('');
    expect(result.current.query).toEqual({ page: 1, page_size: 10 });
  });
});
