CREATE TABLE blog_posts (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    content     TEXT NOT NULL,
    excerpt     TEXT NOT NULL DEFAULT '',
    author_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    published   BOOLEAN NOT NULL DEFAULT FALSE,
    tags        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS blog_posts_author_id_idx  ON blog_posts(author_id);
CREATE INDEX IF NOT EXISTS blog_posts_published_idx  ON blog_posts(published);
CREATE INDEX IF NOT EXISTS blog_posts_created_at_idx ON blog_posts(created_at DESC);
