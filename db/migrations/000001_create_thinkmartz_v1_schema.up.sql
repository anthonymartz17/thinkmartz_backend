CREATE TABLE users (
  id             UUID        PRIMARY KEY,
  email          VARCHAR     NOT NULL UNIQUE,
  username       VARCHAR     NOT NULL UNIQUE,
  password_hash  VARCHAR     NOT NULL,
  bio            TEXT,
  date_of_birth  DATE,
  follower_count INT         NOT NULL DEFAULT 0,
  is_celebrity   BOOLEAN     NOT NULL DEFAULT false,
  created_at     TIMESTAMPTZ NOT NULL,
  updated_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE posts (
  id            UUID        PRIMARY KEY,
  user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content       TEXT        NOT NULL,
  like_count    INT         NOT NULL DEFAULT 0,
  comment_count INT         NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE follows (
  follower_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  followee_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (follower_id, followee_id)
);

CREATE TABLE comments (
  id         UUID        PRIMARY KEY,
  post_id    UUID        NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_id  UUID        REFERENCES comments(id) ON DELETE CASCADE,
  content    TEXT        NOT NULL,
  like_count INT         NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE post_likes (
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id    UUID        NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (user_id, post_id)
);

CREATE TABLE comment_likes (
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  comment_id  UUID        NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_posts_user_created
  ON posts (user_id, created_at DESC);

CREATE INDEX idx_comments_post_created
  ON comments (post_id, created_at DESC);