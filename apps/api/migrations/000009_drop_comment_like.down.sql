-- Recreates the table 000007 made, empty. The likes themselves stay in
-- community; restoring this schema restores none of them.

CREATE TABLE IF NOT EXISTS comment_like (
    post_id    BIGINT      NOT NULL,
    user_id    INTEGER     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS comment_like_user_idx ON comment_like (user_id, post_id);
