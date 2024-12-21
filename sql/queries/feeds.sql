-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFeed :one
-- TODO join on users to get user name?
SELECT * FROM feeds WHERE url = $1;

-- name: GetFeeds :many
-- TODO Is it better to do this join to return the user's name here or just have the application
-- itself make the additional user query to get the name?
SELECT f.id, f.created_at, f.updated_at, f.name, f.url, f.user_id, u.name as user_name
FROM feeds f INNER JOIN users u ON f.user_id = u.id;
