package rand

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateString generates a cryptographically-secure value. If the value is
// unable to be generated an error is returned.
func GenerateString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
