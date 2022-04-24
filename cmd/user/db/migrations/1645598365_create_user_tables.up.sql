CREATE TABLE IF NOT EXISTS users.users (
  id       UUID         NOT NULL DEFAULT gen_random_uuid(),
  email    VARCHAR(320) NOT NULL,
  role     VARCHAR(32)  NOT NULL,

  password BYTEA       NOT NULL,
  salt     VARCHAR(64) NOT NULL,

  verification_hash    VARCHAR(64) NOT  NULL,
  verification_sent_at TIMESTAMP   WITH TIME ZONE NOT NULL,
  verified_at          TIMESTAMP   WITH TIME ZONE,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IdxUsersEmail ON users.users (email);

CREATE TABLE IF NOT EXISTS users.password_resets (
  id                  UUID NOT  NULL DEFAULT gen_random_uuid(),
  user_id             UUID NOT  NULL,
  reset_hash   VARCHAR(64) NOT  NULL,
  requested_at   TIMESTAMP WITH  TIME ZONE NOT NULL,
  completed_at   TIMESTAMP WITH  TIME ZONE,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES users.users (id)
);
