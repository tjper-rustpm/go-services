package config

import "github.com/spf13/viper"

const (
	envPrefix                  = "PAYMENT"
	keyPort                    = "PORT"
	keyDSN                     = "DSN"
	keyMigrations              = "MIGRATIONS"
	keyRedisAddr               = "REDIS_ADDR"
	keyRedisPassword           = "REDIS_PASSWORD"
	keyStripeKey               = "STRIPE_KEY"
	keyStripeWebhookSecret     = "STRIPE_WEBHOOK_SECRET"
	keyStripeMonthlyVIPPriceID = "STRIPE_MONTHLY_VIP_PRICE_ID"
	keyStripeFiveDayVIPPriceID = "STRIPE_FIVE_DAY_VIP_PRICE_ID"
	keyCheckoutEnabled         = "CHECKOUT_ENABLED"
)

func Load() *Config {
	c := &Config{v: viper.New()}
	c.v.SetEnvPrefix(envPrefix)
	c.v.AutomaticEnv()
	c.defaults()

	return c
}

type Config struct {
	v *viper.Viper
}

func (c *Config) defaults() {
	c.v.SetDefault(keyPort, 8080)
	c.v.SetDefault(keyDSN, "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC")
	c.v.SetDefault(keyMigrations, "file:///db/migrations")
	c.v.SetDefault(keyRedisAddr, "redis:6379")
	c.v.SetDefault(keyRedisPassword, "")
	c.v.SetDefault(keyStripeKey, "")
	c.v.SetDefault(keyStripeWebhookSecret, "")
	c.v.SetDefault(keyCheckoutEnabled, true)
	c.v.SetDefault(keyStripeMonthlyVIPPriceID, "price_1MJ7pfCEcXRU8XL2ncBPSC6C")
	c.v.SetDefault(keyStripeFiveDayVIPPriceID, "price_1MJ7ozCEcXRU8XL25CLoBvfA")
}

func (c Config) DSN() string                     { return c.v.GetString(keyDSN) }
func (c Config) Migrations() string              { return c.v.GetString(keyMigrations) }
func (c Config) Port() int                       { return c.v.GetInt(keyPort) }
func (c Config) RedisAddr() string               { return c.v.GetString(keyRedisAddr) }
func (c Config) RedisPassword() string           { return c.v.GetString(keyRedisPassword) }
func (c Config) StripeKey() string               { return c.v.GetString(keyStripeKey) }
func (c Config) StripeWebhookSecret() string     { return c.v.GetString(keyStripeWebhookSecret) }
func (c Config) StripeMonthlyVIPPriceID() string { return c.v.GetString(keyStripeMonthlyVIPPriceID) }
func (c Config) StripeFiveDayVIPPriceID() string { return c.v.GetString(keyStripeFiveDayVIPPriceID) }
func (c Config) CheckoutEnabled() bool           { return c.v.GetBool(keyCheckoutEnabled) }
