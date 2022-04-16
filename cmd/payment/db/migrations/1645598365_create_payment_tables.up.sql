CREATE TABLE IF NOT EXISTS payments.server_subscription_limits (
  server_id UUID     NOT NULL,
  maximum   SMALLINT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (server_id)
);

CREATE UNIQUE INDEX IdxServerSubscriptionLimitsServerID ON payments.server_subscription_limits (server_id);

CREATE TABLE IF NOT EXISTS payments.subscriptions (
  id                     UUID         NOT NULL DEFAULT gen_random_uuid(),
  stripe_checkout_id     VARCHAR(128) NOT NULL,
  stripe_subscription_id VARCHAR(128) NOT NULL,
  stripe_event_id        VARCHAR(128) NOT NULL,

  server_subscription_limit_id SERIAL NOT NULL,
  customer_id                  UUID   NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (server_subscription_limit_id) REFERENCES payments.server_subscription_limits (server_id),
  FOREIGN KEY (customer_id) REFERENCES payments.customers(user_id)
);

CREATE UNIQUE INDEX IdxSubscriptionsStripeSubscriptionID ON payments.subscriptions (stripe_subscription_id);
CREATE INDEX IdxSubscriptionsCustomerID ON payments.subscriptions (customer_id);

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

CREATE TABLE IF NOT EXISTS payments.customers (
  user_id            UUID         NOT NULL,
  steam_id           VARCHAR(128) NOT NULL,
  stripe_customer_id VARCHAR(128) NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (user_id)
)
