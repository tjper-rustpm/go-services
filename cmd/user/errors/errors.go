package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrUserDNE indicates that a process attempted to interact with a user that
	// does not exist.
	ErrUserDNE = errors.New("user dne")

	// ErrPasswordResetDNE indicates that a process attempted to interact with a
	// password reste that does not exist.
	ErrPasswordResetDNE = errors.New("password reset dne")

	// ErrEmailAlreadyInUse indicates that a client attempted to create a user
	// with an email address already being used.
	ErrEmailAlreadyInUse = errors.New("email already in-use")

	// ErrEmailAddressNotRecognized indicates that a user attempted to login, but
	// the email used is not assoiciated with a user.
	ErrEmailAddressNotRecognized = errors.New("email address not recognized")

	// ErrEmailAlreadyVerified indicates that a user attempted to verify their
	// email when it has already been verified.
	ErrEmailAlreadyVerified = errors.New("email already verified")
)

// AsEmailError checks to see if the passed error is of type *EmailError.
func AsEmailError(err error) *EmailError {
	emailErr := new(EmailError)
	if errors.As(err, emailErr) {
		return emailErr
	}
	return nil
}

type EmailError string

func (e EmailError) Error() string {
	return fmt.Sprintf("email invalid; %s", string(e))
}

// AsPasswordError checks to see if the passed error is of type *PasswordError.
func AsPasswordError(err error) *PasswordError {
	passwordError := new(PasswordError)
	if errors.As(err, passwordError) {
		return passwordError
	}
	return nil
}

type PasswordError string

func (e PasswordError) Error() string {
	return fmt.Sprintf("password invalid; %s", string(e))
}

// AsAuthError checks to see if the passed error is of type *AuthError.
func AsAuthError(err error) *AuthError {
	authErr := new(AuthError)
	if errors.As(err, authErr) {
		return authErr
	}
	return nil
}

type AuthError string

func (e AuthError) Error() string {
	return fmt.Sprintf("unauthorized; %s", string(e))
}

// AsHashError checks to see if the passed error is of type *HashError.
func AsHashError(err error) *HashError {
	hashError := new(HashError)
	if errors.As(err, hashError) {
		return hashError
	}
	return nil
}

type HashError string

func (e HashError) Error() string {
	return fmt.Sprintf("hash invalid; %s", string(e))
}
