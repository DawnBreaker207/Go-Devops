import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/renderWithProviders';
import HomePage from '../HomePage';
import type { Movie } from '@/types';

const list = vi.fn();
vi.mock('@/api/movie.api', () => ({ movieApi: { list: (...args: unknown[]) => list(...args) } }));

const movie = (over: Partial<Movie> & { id: string; title: string }): Movie => ({
  genre: 'Sci-Fi',
  duration: 128,
  director: 'Ava Lane',
  description: 'A crew routes humanity through an uncharted nebula.',
  poster_url: `https://cdn.test/${over.id}-poster.jpg`,
  backdrop_url: '',
  trailer_url: '',
  cast: '',
  age_rating: 'T13',
  release_date: '2026-03-01',
  status: 'showing',
  created_at: '2026-01-01T00:00:00+07:00',
  updated_at: '2026-01-01T00:00:00+07:00',
  ...over,
});

const resolveWith = (items: Movie[]) =>
  list.mockResolvedValue({
    items,
    meta: { page: 1, page_size: 100, total: items.length, total_pages: 1 },
  });

beforeEach(() => list.mockReset());

describe('HomePage', () => {
  it('draws each card from the landscape backdrop when the movie has one', async () => {
    resolveWith([
      movie({
        id: 'a',
        title: 'The Dark Horizon',
        backdrop_url: 'https://cdn.test/a-backdrop.jpg',
      }),
    ]);
    renderWithProviders(<HomePage />);

    const card = await screen.findByRole('article');
    // Queried through the DOM, not by role: the card image is `alt=""` on purpose (the title link
    // beside it already names the card), so it is correctly absent from the accessibility tree.
    expect(card.querySelector('img')).toHaveAttribute('src', 'https://cdn.test/a-backdrop.jpg');
  });

  it('falls back to the poster when backdrop_url is empty', async () => {
    // Every movie predating migration 000011 is in this state, so it is the common path today.
    resolveWith([movie({ id: 'b', title: 'Golden Drift' })]);
    renderWithProviders(<HomePage />);

    const card = await screen.findByRole('article');
    expect(card.querySelector('img')).toHaveAttribute('src', 'https://cdn.test/b-poster.jpg');
  });

  it('shows the age rating where the Figma draws a star rating', async () => {
    // The backend has no rating field; rendering "4.5" would be an invented number.
    resolveWith([movie({ id: 'c', title: 'City of Echoes', age_rating: 'T16' })]);
    renderWithProviders(<HomePage />);

    expect(await screen.findByText('T16')).toBeInTheDocument();
    expect(screen.queryByText('4.5')).not.toBeInTheDocument();
  });

  it('hides the Trailers section entirely when no movie has a trailer', async () => {
    resolveWith([movie({ id: 'd', title: 'Last Ticket Home' })]);
    renderWithProviders(<HomePage />);

    await screen.findByRole('article');
    expect(screen.queryByRole('heading', { name: 'Trailer' })).not.toBeInTheDocument();
  });

  it('shows the Trailers section once a movie carries a trailer_url', async () => {
    resolveWith([
      movie({
        id: 'e',
        title: 'Code: Midnight',
        trailer_url: 'https://www.youtube.com/watch?v=YoHD9XEInc0',
      }),
    ]);
    renderWithProviders(<HomePage />);

    expect(await screen.findByRole('heading', { name: 'Trailer' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Phát trailer Code: Midnight' })).toBeInTheDocument();
  });

  it('caps the grid at 8 and reveals the rest through "Show more"', async () => {
    resolveWith(Array.from({ length: 11 }, (_, i) => movie({ id: `m${i}`, title: `Phim ${i}` })));
    const user = userEvent.setup();
    renderWithProviders(<HomePage />);

    await waitFor(() => expect(screen.getAllByRole('article')).toHaveLength(8));

    await user.click(screen.getByRole('button', { name: 'Xem thêm phim' }));

    expect(screen.getAllByRole('article')).toHaveLength(11);
    // Nothing left to reveal, so the button goes away rather than sitting there inert.
    expect(screen.queryByRole('button', { name: 'Xem thêm phim' })).not.toBeInTheDocument();
  });

  it('asks the API only once: "Show more" pages the list it already has', async () => {
    resolveWith(Array.from({ length: 11 }, (_, i) => movie({ id: `m${i}`, title: `Phim ${i}` })));
    const user = userEvent.setup();
    renderWithProviders(<HomePage />);

    await waitFor(() => expect(screen.getAllByRole('article')).toHaveLength(8));
    await user.click(screen.getByRole('button', { name: 'Xem thêm phim' }));

    expect(list).toHaveBeenCalledTimes(1);
  });

  it('keeps drafts and ended films off the customer home', async () => {
    resolveWith([
      movie({ id: 'showing', title: 'Dang chieu' }),
      movie({ id: 'draft', title: 'Ban nhap', status: 'draft' }),
      movie({ id: 'ended', title: 'Da ket thuc', status: 'ended' }),
    ]);
    renderWithProviders(<HomePage />);

    // findAllByText, not findByText: the hero shows the same title as the card it came from.
    expect(await screen.findAllByText('Dang chieu')).not.toHaveLength(0);
    expect(screen.queryByText('Ban nhap')).not.toBeInTheDocument();
    expect(screen.queryByText('Da ket thuc')).not.toBeInTheDocument();
    expect(screen.getAllByRole('article')).toHaveLength(1);
  });
});
