package config

import "github.com/spf13/viper"

const (
	envPrefix        = "PAYMENT"
	keyPort          = "PORT"
	keyDSN           = "DSN"
	keyMigrations    = "MIGRATIONS"
	keyStripeKey     = "STRIPE_KEY"
	keyRedisAddr     = "REDIS_ADDR"
	keyRedisPassword = "REDIS_PASSWORD"
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
	c.v.SetDefault(keyStripeKey, "")
	c.v.SetDefault(keyRedisAddr, "redis:6379")
	c.v.SetDefault(keyRedisPassword, "")
}

func (c Config) DSN() string           { return c.v.GetString(keyDSN) }
func (c Config) Migrations() string    { return c.v.GetString(keyMigrations) }
func (c Config) Port() int             { return c.v.GetInt(keyPort) }
func (c Config) StripeKey() string     { return c.v.GetString(keyStripeKey) }
func (c Config) RedisAddr() string     { return c.v.GetString(keyRedisAddr) }
func (c Config) RedisPassword() string { return c.v.GetString(keyRedisPassword) }
