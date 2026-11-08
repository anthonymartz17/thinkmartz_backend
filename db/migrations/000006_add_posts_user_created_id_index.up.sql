DROP INDEX idx_posts_user_created;

CREATE INDEX idx_posts_user_created
  ON posts (user_id, created_at DESC, id DESC);
