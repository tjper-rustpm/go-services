-- name: CreateServer :one
INSERT INTO rustpm.server (
  name,
  instance_id,
  instance_kind,
  allocation_id,
  elastic_ip,
  max_players,
  map_size,
  map_seed,
  map_salt,
  tick_rate,
  rcon_password,
  description,
  url,
  background,
  banner_url,
  wipe_day,
  blueprint_wipe_frequency,
  map_wipe_frequency,
  region
)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
RETURNING id;

-- name: GetServer :one
SELECT *
FROM rustpm.server
WHERE id = $1;

-- name: ListLiveServerDefinitions :many
SELECT *
FROM rustpm.server
WHERE EXISTS (
  SELECT 1
  FROM rustpm.live_server
  WHERE live_server.id = server.id
);

-- name: ListDormantServerDefinitions :many
SELECT *
FROM rustpm.server
WHERE EXISTS (
  SELECT 1
  FROM rustpm.dormant_server
  WHERE dormant_server.id = server.id
);

-- name: ListArchivedServerDefinitions :many
SELECT *
FROM rustpm.server
WHERE EXISTS (
  SELECT 1
  FROM rustpm.archived_server
  WHERE archived_server.id = server.id
);

-- name: ListLiveServers :many
SELECT live_server.*
FROM rustpm.live_server;

-- name: ListDormantServers :many
SELECT dormant_server.*
FROM rustpm.dormant_server;

-- name: ListArchivedServers :many
SELECT archived_server.*
FROM rustpm.archived_server;

-- name: ArchiveServer :exec
INSERT INTO rustpm.archived_server (id)
VALUES($1);

-- name: CreateDormantServer :exec
INSERT INTO rustpm.dormant_server (id)
VALUES($1);

-- name: CreateLiveServer :exec
INSERT INTO rustpm.live_server (
  id,
  association_id,
  active_players,
  queued_players
)
VALUES($1, $2, $3, $4);

-- name: UpdateLiveServer :exec
UPDATE rustpm.live_server
SET active_players = $1,
    queued_players = $2
WHERE live_server.id = $3;

-- name: GetLiveServer :one
SELECT
  live_server.*,
  server.*
FROM
  rustpm.live_server
JOIN
  rustpm.server
  ON server.id = live_server.id
WHERE
  live_server.id = $1;

-- name: GetLiveServerDetails :one
SELECT live_server.*
FROM rustpm.live_server
WHERE live_server.id = $1;

-- name: GetDormantServer :one
SELECT
  dormant_server.*,
  server.*
FROM
  rustpm.dormant_server
JOIN
  rustpm.server
  ON server.id = dormant_server.id
WHERE
  dormant_server.id = $1;

-- name: GetDormantServerDetails :one
SELECT dormant_server.*
FROM rustpm.dormant_server
WHERE dormant_server.id = $1;

-- name: RemoveLiveServer :exec
DELETE FROM rustpm.live_server
WHERE live_server.id = $1;

-- name: RemoveDormantServer :exec
DELETE FROM rustpm.dormant_server
WHERE dormant_server.id = $1;

-- name: GetServerRconInfo :one
SELECT
  server.rcon_password,
  server.elastic_ip
FROM rustpm.server
WHERE server.id = $1;
