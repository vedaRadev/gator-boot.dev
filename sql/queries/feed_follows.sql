-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    inserted_feed_follow.*,
    f.name AS feed_name,
    u.name AS user_name
FROM inserted_feed_follow
INNER JOIN users u ON u.id = inserted_feed_follow.user_id
INNER JOIN feeds f ON f.id = inserted_feed_follow.feed_id;

-- name: GetFeedFollowsForUser :many
SELECT ff.*, f.url AS feed_url, f.name AS feed_name
FROM feed_follows ff
INNER JOIN feeds f ON ff.feed_id = f.id
WHERE $1 = ff.user_id;
