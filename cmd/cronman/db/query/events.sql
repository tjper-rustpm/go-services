-- name: GetServerEvent :one
SELECT server_event.*
FROM rustpm.server_event
WHERE server_event.id = $1;

-- name: GetServerEvents :many
SELECT server_event.*
FROM rustpm.server_event
WHERE server_event.server_id = $1;


-- name: ListActiveServerEvents :many
SELECT *
FROM
  rustpm.server_event
WHERE NOT EXISTS (
  SELECT 1
  FROM rustpm.archived_server
  WHERE archived_server.id = server_event.server_id
);

-- name: ListServerEvents :many
SELECT *
FROM
  rustpm.server_event
WHERE server_event.server_id = ANY($1::UUID[]);

-- name: CreateServerEvent :one
INSERT INTO rustpm.server_event (
  server_id,
  day,
  hour,
  kind
)
VALUES($1, $2, $3, $4)
RETURNING id;

-- name: RemoveServerEvent :exec
DELETE FROM rustpm.server_event
WHERE id = $1;

-- name: RemoveServerEvents :exec
DELETE FROM rustpm.server_event
WHERE server_event.server_id = $1
      AND server_event.id = ANY($2::UUID[]);
