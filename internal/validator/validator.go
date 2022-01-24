package validator

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// New creates a new validator instance.
func New() *validator.Validate {
	valid := validator.New()
	if err := RegisterPasswordValidation(valid); err != nil {
		panic(fmt.Sprintf("validator initialization; error: %s", err))
	}
	if err := RegisterCronValidation(valid); err != nil {
		panic(fmt.Sprintf("validator initialization; error: %s", err))
	}

	return valid
}

// RegisterPasswordValidation registers the "password" field validator with the
// validator instance.
func RegisterPasswordValidation(validator *validator.Validate) error {
	return validator.RegisterValidation("password", password)
}

var (
	passwordRE            = regexp.MustCompile(`^[a-zA-Z\d \!\"\#\$\%\&\'\(\)\*\+\,\-\.\/\:\;\<\=\>\?\@\[\]\^\_\x60\{\|\}\~]{8,64}$`)
	atLeastOneLowerCaseRE = regexp.MustCompile(`[a-z]+`)
	atLeastOneUpperCaseRE = regexp.MustCompile(`[A-Z]+`)
	atLeastOneNumberRE    = regexp.MustCompile(`[\d]+`)
)

// password matches against strings that satisfy the following requirements:
// - between 8 and 64 characters in length
// - at least one lower-case letter
// - at least one upper-case letter
// - at least one number
// - special characters are allowed
// In the event there is not a match, the reason is returned as an error.
func password(fl validator.FieldLevel) bool {
	const (
		minLength = 8
		maxLength = 64
	)
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	switch {
	case len(val) < minLength:
		return false
	case len(val) > maxLength:
		return false
	case !passwordRE.MatchString(val):
		return false
	case !atLeastOneLowerCaseRE.MatchString(val):
		return false
	case !atLeastOneUpperCaseRE.MatchString(val):
		return false
	case !atLeastOneNumberRE.MatchString(val):
		return false
	}
	return true
}

// RegisterCronValidation registers the "cron" field validator with the
// validator instance.
func RegisterCronValidation(validator *validator.Validate) error {
	return validator.RegisterValidation("cron", cron)
}

var cronRE = regexp.MustCompile(`^([\*]|[0-9]|[1-5][0-9])\s([\*]|[0-9]|[1][0-9]|[2][0-3])\s([\*]|([1-9]|[12][0-9]|[3][01])|(([1-9]|[12][0-9]|[3][01])-([1-9]|[12][0-9]|[3][01])))\s[\*\?]\s[\*\?]$`)

// cron matches against strings that specify a cron schedule. Examples are:
// - 0 22 8-14 * ?
// - 0 20 * * ?
func cron(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	return cronRE.MatchString(val)
}
