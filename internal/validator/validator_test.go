package validator

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	tests := map[string]struct {
		value interface{}
		tag   string
		err   bool
	}{
		"password too short":               {value: "1ab!", tag: "password", err: true},
		"password too long":                {value: strings.Repeat("1aB!", 17), tag: "password", err: true},
		"password at least one lower-case": {value: "1AB!CDEF", tag: "password", err: true},
		"password at least one upper-case": {value: "1ab!cdef", tag: "password", err: true},
		"password at least one number":     {value: "aBc!defg", tag: "password", err: true},
		"valid password":                   {value: "1ValidPassword!", tag: "password", err: false},
		"invalid password":                 {value: "invalid-password", tag: "password", err: true},
		"daily cron w/ minute":             {value: "30 21 * * *", tag: "cron", err: false},
		"daily cron w/o minute":            {value: "0 21 * * *", tag: "cron", err: false},
		"1st week cron":                    {value: "0 21 1-7 * *", tag: "cron", err: false},
		"2nd week cron":                    {value: "0 21 8-14 * *", tag: "cron", err: false},
		"3rd week cron":                    {value: "0 21 15-21 * *", tag: "cron", err: false},
		"4th week cron":                    {value: "0 21 22-28 * *", tag: "cron", err: false},
		"5th week cron":                    {value: "0 21 29-31 * *", tag: "cron", err: false},
		"6th week cron, too many days":     {value: "0 21 29-35 * *", tag: "cron", err: true},
		"daily, too many minutes":          {value: "60 21 * * *", tag: "cron", err: true},
		"daily, too many hours":            {value: "0 24 * * *", tag: "cron", err: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			valid := New()
			err := valid.Var(&test.value, test.tag)
			if test.err {
				errors := make(validator.ValidationErrors, 0)
				assert.ErrorAs(t, err, &errors)

				for _, err := range errors {
					assert.Equal(t, err.Tag(), test.tag)
				}
				return
			}
			assert.Nil(t, err)
		})
	}
}
