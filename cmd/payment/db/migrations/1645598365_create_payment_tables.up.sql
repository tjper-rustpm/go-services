CREATE TABLE IF NOT EXISTS payments.servers (
  id                 UUID     NOT NULL,
  subscription_limit SMALLINT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS payments.customers (
  id                 UUID         NOT NULL DEFAULT gen_random_uuid(),
  steam_id           VARCHAR(128) NOT NULL,
  stripe_customer_id VARCHAR(128) NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IdxCustomersSteamID ON payments.customers (steam_id);
CREATE UNIQUE INDEX IdxCustomersStripeCustomerID ON payments.customers (stripe_customer_id);

CREATE TABLE IF NOT EXISTS payments.users (
  id          UUID NOT NULL,
  customer_id UUID NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (customer_id) references payments.customers (id)
);

CREATE UNIQUE INDEX IdxUsersCustomerID ON payments.users (customer_id);

CREATE TABLE IF NOT EXISTS payments.vips (
  id                 UUID         NOT NULL DEFAULT gen_random_uuid(),
  stripe_checkout_id VARCHAR(128) NOT NULL,
  stripe_event_id    VARCHAR(128) NOT NULL,

  server_id   UUID NOT NULL,
  customer_id UUID NOT NULL,

  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_id) REFERENCES payments.servers (id),
  FOREIGN KEY (customer_id) REFERENCES payments.customers(id)
);

CREATE INDEX IdxVipsCustomerID ON payments.vips (customer_id);
CREATE UNIQUE INDEX IdxVipsStripeEventID ON payments.vips (stripe_event_id);

CREATE TABLE IF NOT EXISTS payments.subscriptions (
  id                     UUID         NOT NULL DEFAULT gen_random_uuid(),
  vip_id            UUID         NOT NULL,
  stripe_subscription_id VARCHAR(128) NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (vip_id) REFERENCES payments.vips (id)
);

CREATE UNIQUE INDEX IdxSubscriptionsStripeSubscriptionID ON payments.subscriptions (stripe_subscription_id);

CREATE TABLE IF NOT EXISTS payments.invoices (
  id              UUID         NOT NULL DEFAULT gen_random_uuid(),
  subscription_id UUID         NOT NULL,
  stripe_event_id VARCHAR(128) NOT NULL,
  status          VARCHAR(64)  NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (subscription_id) REFERENCES payments.subscriptions (id)
);

CREATE UNIQUE INDEX IdxInvoicesStripeEventID ON payments.invoices (stripe_event_id);
