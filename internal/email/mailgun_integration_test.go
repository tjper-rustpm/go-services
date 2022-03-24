// +build mailgunintegration

package email

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tjper/rustcron/cmd/user/config"

	"github.com/mailgun/mailgun-go/v4"
)

func TestSendPasswordReset(t *testing.T) {
	tests := map[string]struct {
		to   string
		hash string
	}{
		"rustcron": {to: "rustcron@gmail.com", hash: "fakehash"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cfg := config.Load()

			mg := mailgun.NewMailgun(cfg.MailgunDomain(), cfg.MailgunAPIKey())
			emailer := NewMailgunEmailer(mg, cfg.MailgunHost())

			err := emailer.SendPasswordReset(ctx, test.to, test.hash)
			require.Nil(t, err)
		})
	}
}

func TestSendVerifyEmail(t *testing.T) {
	tests := map[string]struct {
		to   string
		hash string
	}{
		"rustcron": {to: "rustcron@gmail.com", hash: "fakehash"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cfg := config.Load()

			mg := mailgun.NewMailgun(cfg.MailgunDomain(), cfg.MailgunAPIKey())
			emailer := NewMailgunEmailer(mg, cfg.MailgunHost())

			err := emailer.SendVerifyEmail(ctx, test.to, test.hash)
			require.Nil(t, err)
		})
	}
}
