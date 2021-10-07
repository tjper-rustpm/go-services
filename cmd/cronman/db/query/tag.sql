-- name: GetServerTags :many
SELECT *
FROM rustpm.server_tag
WHERE server_tag.server_id = $1;

-- name: ListServerTags :many
SELECT *
FROM
  rustpm.server_tag
WHERE
  server_tag.server_id = ANY($1::UUID[]);

-- name: CreateServerTag :one
INSERT INTO rustpm.server_tag(
  server_id,
  description,
  icon,
  value
)
VALUES($1, $2, $3, $4)
RETURNING id;

-- name: RemoveServerTag :exec
DELETE FROM rustpm.server_tag
WHERE server_tag.id = $1;

-- name: RemoveServerTags :exec
DELETE FROM rustpm.server_tag
WHERE server_tag.server_id = $1
      AND server_tag.id = ANY($2::UUID[]);
