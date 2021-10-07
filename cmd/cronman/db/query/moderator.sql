-- name: CreateServerModerator :one
INSERT INTO rustpm.server_moderator (server_id, steam_id)
VALUES($1, $2)
RETURNING id;

-- name: GetServerModerator :one
SELECT *
FROM
  rustpm.server_moderator
WHERE
  server_moderator.id = $1;

-- name: ListServerModerators :many
SELECT *
FROM
  rustpm.server_moderator
WHERE
  server_moderator.server_id = ANY($1::UUID[]);


-- name: GetServerModerators :many
SELECT *
FROM
  rustpm.server_moderator
WHERE
  server_moderator.server_id = $1
  AND NOT EXISTS (
    SELECT 1
    FROM rustpm.server_removed_moderator
    WHERE server_removed_moderator.id = server_moderator.id
  );

-- name: GetServerModeratorsPendingRemoval :many
SELECT *
FROM
  rustpm.server_moderator
WHERE
  server_moderator.server_id = $1
  AND EXISTS (
    SELECT 1
    FROM rustpm.server_removed_moderator
    WHERE server_removed_moderator.id = server_moderator.id
          AND enacted = FALSE
  );

-- name: EnactServerModeratorRemoval :exec
UPDATE rustpm.server_removed_moderator
SET enacted = TRUE,
    enacted_at = now() AT TIME ZONE 'UTC'
WHERE server_removed_moderator.id = $1;

-- name: MarkServerModeratorForRemoval :exec
INSERT INTO rustpm.server_removed_moderator (id, enacted)
VALUES($1, FALSE);

-- name: RemoveServerModerator :exec
INSERT INTO rustpm.server_removed_moderator (id, enacted, enacted_at)
VALUES($1, TRUE, now() AT TIME ZONE 'UTC');
