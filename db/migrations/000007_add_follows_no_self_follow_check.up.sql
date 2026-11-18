ALTER TABLE follows
  ADD CONSTRAINT follows_no_self_follow CHECK (follower_id <> followee_id);
