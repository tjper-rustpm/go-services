package config

import (
	"net/http"

	"github.com/spf13/viper"
)

const (
	envPrefix         = "USER"
	keyPort           = "PORT"
	keyDSN            = "DSN"
	keyMigrations     = "MIGRATIONS"
	keyRedisAddr      = "REDIS_ADDR"
	keyRedisPassword  = "REDIS_PASSWORD"
	keyMailgunDomain  = "MAILGUN_DOMAIN"
	keyMailgunAPIKey  = "MAILGUN_API_KEY"
	keyMailgunHost    = "MAILGUN_HOST"
	keyCookieDomain   = "COOKIE_DOMAIN"
	keyCookieSecure   = "COOKIE_SECURE"
	keyCookieSameSite = "COOKIE_SAMESITE"
	keyAdmins         = "ADMINS"
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
	c.viper.SetDefault(keyMailgunDomain, "mg.rustpm.com")
	c.viper.SetDefault(keyMailgunAPIKey, "")
	c.viper.SetDefault(keyMailgunHost, "http://localhost")
	c.viper.SetDefault(keyCookieDomain, "localhost")
	c.viper.SetDefault(keyCookieSecure, false)
	c.viper.SetDefault(keyCookieSameSite, "off")
	c.viper.SetDefault(keyAdmins, []string{})
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

func MailgunDomain() string {
	return global.viper.GetString(keyMailgunDomain)
}

func MailgunAPIKey() string {
	return global.viper.GetString(keyMailgunAPIKey)
}

func MailgunHost() string {
	return global.viper.GetString(keyMailgunHost)
}

func CookieDomain() string {
	return global.viper.GetString(keyCookieDomain)
}

func CookieSecure() bool {
	return global.viper.GetBool(keyCookieSecure)
}

func CookieSameSite() http.SameSite {
	sameSiteStr := global.viper.GetString(keyCookieSameSite)

	var sameSite http.SameSite
	switch sameSiteStr {
	case "off":
		sameSite = http.SameSiteDefaultMode
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	default:
		panic("unrecognized Same-Site cookie configuration value")
	}

	return sameSite
}

func Admins() []string {
	return global.viper.GetStringSlice(keyAdmins)
}
