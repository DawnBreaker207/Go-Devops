-- Module article: news/promotion CMS. Public reads only ever see
-- status=published (enforced in the service, not here); operators manage
-- drafts through /admin/articles. Slug uniqueness, view counter and soft
-- delete are kept as is.

CREATE TABLE IF NOT EXISTS articles (
    id            UUID         PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL,
    summary       VARCHAR(500),
    thumbnail_url VARCHAR(1024),
    content       TEXT         NOT NULL,
    author_id     UUID         NOT NULL REFERENCES users(id),
    type          VARCHAR(16)  NOT NULL DEFAULT 'news',
    status        VARCHAR(16)  NOT NULL DEFAULT 'draft',
    views         BIGINT       NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT ck_article_type CHECK (type IN ('news','promotion')),
    CONSTRAINT ck_article_status CHECK (status IN ('draft','published','hidden'))
);

-- Partial unique index rather than a plain UNIQUE: a soft-deleted article
-- frees its slug for reuse afterwards.
CREATE UNIQUE INDEX IF NOT EXISTS idx_articles_slug
    ON articles (slug) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_status_created
    ON articles (status, created_at);
