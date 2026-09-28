/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string;
  readonly VITE_APP_NAME: string;
  /** TMDB v4 read-only Bearer token; empty means the TMDB lookup stays hidden. */
  readonly VITE_TMDB_READ_TOKEN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
