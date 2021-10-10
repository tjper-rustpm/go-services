package controller

import (
	"strings"
	"testing"

	rpmerrors "github.com/tjper/rustcron/cmd/user/errors"

	"github.com/stretchr/testify/assert"
)

func TestIsPassword(t *testing.T) {
	tests := map[string]struct {
		password string
		err      error
	}{
		"too short": {
			password: "6Chars",
			err:      rpmerrors.PasswordError("minimum of 8 characters"),
		},
		"too long": {
			password: "TooManyChars" + strings.Repeat("1", 64),
			err:      rpmerrors.PasswordError("maximum of 64 characters"),
		},
		"no lower-case letter": {
			password: "1NOLOWERCASE",
			err:      rpmerrors.PasswordError("at least one lower-case letter required"),
		},
		"no upper-case letter": {
			password: "1nouppercase",
			err:      rpmerrors.PasswordError("at least one upper-case letter required"),
		},
		"no number": {
			password: "NoNumber",
			err:      rpmerrors.PasswordError("at least one number required"),
		},
		"valid password":                {password: "1ValidPassword", err: nil},
		"valid with special characters": {password: "1ValidPassword!", err: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.err, validatePassword(test.password))
		})
	}
}
