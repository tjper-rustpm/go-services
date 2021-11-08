package email

import (
	"context"
	"fmt"

	"github.com/mailgun/mailgun-go/v4"
)

func NewMailgunEmailer(mg *mailgun.MailgunImpl, host string) *MailgunEmailer {
	return &MailgunEmailer{
		mg:   mg,
		host: host,
	}
}

// MailgunEmailer is responsibel for mailgun API interactions.
type MailgunEmailer struct {
	mg   *mailgun.MailgunImpl
	host string
}

const (
	resetPassword = "reset_password"
	verifyEmail   = "verify_email"
)

// SendPasswordReset sends a reset_password email to the "to" email specified.
// Mailgun templates are used, acquire access to the Mailgun UI to learn more.
func (e MailgunEmailer) SendPasswordReset(ctx context.Context, to, hash string) error {
	msg := e.mg.NewMessage("password_reset@mg.rustpm.com", "Forgot your password?", "", to)
	msg.SetTemplate(resetPassword)
	if err := addResetPasswordURL(msg, e.host, hash); err != nil {
		return err
	}

	return e.send(ctx, msg)
}

// SendVerifyEmail sends a verify_email email to the the "to" email specified.
// Mailgun templates are used, acquire access to the Mailgun UI to learn more.
func (e MailgunEmailer) SendVerifyEmail(ctx context.Context, to, hash string) error {
	msg := e.mg.NewMessage("verify-email@mg.rustpm.com", "Verify your email.", "", to)
	msg.SetTemplate(verifyEmail)
	if err := addVerifyEmailURL(msg, e.host, hash); err != nil {
		return err
	}

	return e.send(ctx, msg)
}

// --- private ---

func (e MailgunEmailer) send(ctx context.Context, msg *mailgun.Message) error {
	if _, _, err := e.mg.Send(ctx, msg); err != nil {
		return err
	}
	return nil
}

// --- helper ---

func addVerifyEmailURL(msg *mailgun.Message, host, hash string) error {
	return msg.AddTemplateVariable(
		"verifyEmailURL",
		fmt.Sprintf("https://%s/verify-email?hash=%s", host, hash),
	)
}

func addResetPasswordURL(msg *mailgun.Message, host, hash string) error {
	return msg.AddTemplateVariable(
		"resetPasswordURL",
		fmt.Sprintf("https://%s/user/reset-password?hash=%s", host, hash),
	)
}
