-- Seed data for manual testing (Postman, etc.) against the docker-compose
-- Postgres instance. Run with `make seed`; undo with `make seed-down`.
--
-- Creates 4 seed users (alice, bob, carol, celeb — password "Seed1234!" for
-- all of them), a follow cluster among them, follows from the real account
-- anthonymartz17@gmail.com onto bob/carol/celeb (so GET /feed returns
-- something once logged in as that account), and a handful of posts with
-- staggered timestamps. `celeb` gets an artificially huge follower_count so
-- its posts exercise feed's celebrity merge-at-read-time path instead of
-- normal fan-out.

INSERT INTO users (email, username, password_hash)
VALUES ('alice@seed.example', 'alice', '$2a$10$HBl1Eb3/LQHmbiD/3UZuX.JRpGaEyQOF1s4eKK9.PhBXaP4Gz5hyO')
RETURNING id AS alice_id \gset

INSERT INTO users (email, username, password_hash)
VALUES ('bob@seed.example', 'bob', '$2a$10$HBl1Eb3/LQHmbiD/3UZuX.JRpGaEyQOF1s4eKK9.PhBXaP4Gz5hyO')
RETURNING id AS bob_id \gset

INSERT INTO users (email, username, password_hash)
VALUES ('carol@seed.example', 'carol', '$2a$10$HBl1Eb3/LQHmbiD/3UZuX.JRpGaEyQOF1s4eKK9.PhBXaP4Gz5hyO')
RETURNING id AS carol_id \gset

-- follower_count is set directly here; the follows-trigger only fires on
-- inserts/deletes into the follows table, so this sticks unless real
-- follows rows targeting celeb are added afterward (they are, below, but
-- the trigger's +1s just add on top of this baseline).
INSERT INTO users (email, username, password_hash, follower_count)
VALUES ('celeb@seed.example', 'celeb', '$2a$10$HBl1Eb3/LQHmbiD/3UZuX.JRpGaEyQOF1s4eKK9.PhBXaP4Gz5hyO', 5000000)
RETURNING id AS celeb_id \gset

SELECT id AS real_user_id FROM users WHERE email = 'anthonymartz17@gmail.com' \gset

-- Follow cluster among the seed users themselves.
INSERT INTO follows (follower_id, followee_id) VALUES
	(:'alice_id', :'bob_id'),
	(:'alice_id', :'carol_id'),
	(:'alice_id', :'celeb_id'),
	(:'bob_id',   :'alice_id'),
	(:'bob_id',   :'celeb_id'),
	(:'carol_id', :'alice_id'),
	(:'carol_id', :'bob_id');

-- The real registered account follows the seed users, so GET /feed shows
-- their posts once logged in as anthonymartz17@gmail.com.
INSERT INTO follows (follower_id, followee_id) VALUES
	(:'real_user_id', :'bob_id'),
	(:'real_user_id', :'carol_id'),
	(:'real_user_id', :'celeb_id');

-- Posts, staggered a few hours apart so feed ordering is visually obvious.
INSERT INTO posts (user_id, content, created_at) VALUES
	(:'bob_id',   'bob: just setting up my thinkmartz',              now() - interval '6 hours'),
	(:'bob_id',   'bob: second post, a bit later',                   now() - interval '5 hours'),
	(:'carol_id', 'carol: hello world from carol',                   now() - interval '4 hours'),
	(:'celeb_id', 'celeb: big announcement incoming',                now() - interval '3 hours'),
	(:'alice_id', 'alice: this one should NOT show up in your feed (you do not follow alice)', now() - interval '2 hours 30 minutes'),
	(:'bob_id',   'bob: one more for good measure',                  now() - interval '2 hours'),
	(:'celeb_id', 'celeb: follow-up post',                            now() - interval '1 hour'),
	(:'carol_id', 'carol: freshest post in the feed',                 now() - interval '10 minutes');

-- Filler posts, deliberately older than everything above, to push the total
-- followee-visible post count past _feedPageSize (20) so GET /feed without a
-- cursor returns a next_cursor and a second page is testable. 12 (bob) + 8
-- (carol) + the 7 non-filler bob/carol/celeb posts above = 27 total.
INSERT INTO posts (user_id, content, created_at)
SELECT :'bob_id', 'bob: filler post #' || i, now() - interval '1 day' - (i || ' hours')::interval
FROM generate_series(1, 12) AS i;

INSERT INTO posts (user_id, content, created_at)
SELECT :'carol_id', 'carol: filler post #' || i, now() - interval '1 day' - (i || ' hours')::interval
FROM generate_series(1, 8) AS i;
