CREATE TABLE IF NOT EXISTS payment.subscriptions (
  id                     UUID         NOT NULL DEFAULT gen_random_uuid(),
  stripe_checkout_id     VARCHAR(128) NOT NULL,
  stripe_customer_id     VARCHAR(128) NOT NULL,
  stripe_subscription_id VARCHAR(128) NOT NULL,
  stripe_event_id        VARCHAR(128) NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IdxSubscriptionsStripeSubscriptionID ON payment.subscriptions (stripe_subscription_id);

CREATE TABLE IF NOT EXISTS payment.invoices (
  id              UUID         NOT NULL DEFAULT gen_random_uuid(),
  subscription_id UUID         NOT NULL,
  stripe_event_id VARCHAR(128) NOT NULL,
  status          VARCHAR(64)  NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (id),
  FOREIGN KEY (subscription_id) REFERENCES payment.subscriptions (id)
);
