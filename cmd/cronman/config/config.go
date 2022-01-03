package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	envPrefix           = "CRONMAN"
	keyPort             = "PORT"
	keyDSN              = "DSN"
	keyMigrations       = "MIGRATIONS"
	keyRedisAddr        = "REDIS_ADDR"
	keyRedisPassword    = "REDIS_PASSWORD"
	keyDirectorEnabled  = "DIRECTOR_ENABLED"
	keyHttpReadTimeout  = "HTTP_READ_TIMEOUT"
	keyHttpWriteTimeout = "HTTP_WRITE_TIMEOUT"
)

var global *config

func init() {
	c := &config{
		viper: viper.New(),
	}
	c.viper.SetEnvPrefix(envPrefix)
	c.viper.AutomaticEnv()
	c.loadDefaults()
	global = c
}

type config struct {
	viper *viper.Viper
}

func (c *config) loadDefaults() {
	c.viper.SetDefault(keyPort, 8080)
	c.viper.SetDefault(keyDSN, "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC")
	c.viper.SetDefault(keyMigrations, "file:///db/migrations")
	c.viper.SetDefault(keyRedisAddr, "redis:6379")
	c.viper.SetDefault(keyRedisPassword, "")
	c.viper.SetDefault(keyDirectorEnabled, false)
	c.viper.SetDefault(keyHttpReadTimeout, 500*time.Millisecond)
	c.viper.SetDefault(keyHttpWriteTimeout, 30*time.Minute)
}

func Port() int {
	return global.viper.GetInt(keyPort)
}

func DSN() string {
	return global.viper.GetString(keyDSN)
}

func Migrations() string {
	return global.viper.GetString(keyMigrations)
}

func RedisAddr() string {
	return global.viper.GetString(keyRedisAddr)
}

func RedisPassword() string {
	return global.viper.GetString(keyRedisPassword)
}

func DirectorEnabled() bool {
	return global.viper.GetBool(keyDirectorEnabled)
}

func HttpReadTimeout() time.Duration {
	return global.viper.GetDuration(keyHttpReadTimeout)
}

func HttpWriteTimeout() time.Duration {
	return global.viper.GetDuration(keyHttpWriteTimeout)
}
