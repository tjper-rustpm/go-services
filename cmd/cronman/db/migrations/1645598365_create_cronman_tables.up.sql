CREATE TABLE IF NOT EXISTS servers.servers (
  id            UUID     NOT NULL DEFAULT gen_random_uuid(),
  name          VARCHAR  NOT NULL,
  instance_id   VARCHAR  NOT NULL,
  instance_kind VARCHAR  NOT NULL,
  allocation_id VARCHAR  NOT NULL,
  elastic_ip    VARCHAR  NOT NULL,
  max_players   SMALLINT NOT NULL,
  map_size      SMALLINT NOT NULL,
  tick_rate     SMALLINT NOT NULL,
  rcon_password VARCHAR  NOT NULL,
  description   VARCHAR  NOT NULL,
  background    VARCHAR  NOT NULL,
  url           VARCHAR  NOT NULL,
  banner_url    VARCHAR  NOT NULL,
  region        VARCHAR  NOT NULL,
  options       JSONB    NOT NULL default '{}'::JSONB,

  state_id   UUID    NOT NULL,
  state_type VARCHAR NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS servers.live_servers (
  id UUID NOT NULL DEFAULT gen_random_uuid(),

  association_id VARCHAR NOT NULL,
  active_players SMALLINT NOT NULL,
  queued_players SMALLINT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS servers.dormant_servers (
  id UUID NOT NULL DEFAULT gen_random_uuid(),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS servers.archived_servers (
  id UUID NOT NULL DEFAULT gen_random_uuid(),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS servers.wipes (
  id        UUID NOT NULL DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL,

  map_seed SMALLINT NOT NULL,
  map_salt SMALLINT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES servers.servers (id)
);

CREATE TABLE IF NOT EXISTS servers.tags (
  id        UUID NOT NULL DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL,

  description VARCHAR NOT NULL,
  icon        VARCHAR NOT NULL,
  value       VARCHAR NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES servers.servers (id)
);

CREATE TABLE IF NOT EXISTS servers.moderators (
  id        UUID NOT NULL DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL,

  steam_id VARCHAR NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES servers.servers (id)
);

CREATE TABLE IF NOT EXISTS servers.events (
  id        UUID NOT NULL DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL,

  schedule VARCHAR NOT NULL,
  weekday  SMALLINT,
  kind     VARCHAR NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES servers.servers (id)
);

CREATE TABLE IF NOT EXISTS servers.vips (
  id              UUID         NOT NULL DEFAULT gen_random_uuid(),
  server_id       UUID         NOT NULL,
  subscription_id UUID         NOT NULL,
  steam_ID        VARCHAR(128) NOT NULL,

  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES servers.servers (id)
);
