package config

import "github.com/spf13/viper"

const (
	envPrefix    = "PAYMENT"
	keyStripeKey = "STRIPE_KEY"
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
	c.v.SetDefault(keyStripeKey, "")
}

func (c Config) StripeKey() string {
	return c.v.GetString(keyStripeKey)
}
