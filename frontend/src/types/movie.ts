export type MovieStatus = 'draft' | 'showing' | 'ended';

export interface Movie {
  id: string;
  title: string;
  genre: string;
  duration: number;
  director: string;
  description: string;
  poster_url: string;
  release_date: string;
  status: MovieStatus;
  created_at: string;
  updated_at: string;
}

export interface MoviePayload {
  title: string;
  genre: string;
  duration: number;
  director: string;
  description?: string;
  poster_url?: string;
  release_date: string;
  status: MovieStatus;
}
