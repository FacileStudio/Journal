package authcrypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// NewToken returns 32 random bytes URL-base64-encoded, for API keys and
// credentials that must not be guessable.
func NewToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken returns the SHA-256 hex of a token, the form stored in the
// database so a leaked table never discloses a usable key.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// BearerToken extracts the token from an Authorization header, returning
// false unless it carried a well-formed bearer scheme and a non-empty value.
func BearerToken(authorization string) (string, bool) {
	const scheme = "bearer "
	if len(authorization) <= len(scheme) || !strings.EqualFold(authorization[:len(scheme)], scheme) {
		return "", false
	}
	token := strings.TrimSpace(authorization[len(scheme):])
	return token, token != ""
}
