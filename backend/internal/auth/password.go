// Package auth implements password hashing, signed-cookie sessions, TOTP MFA,
// and an HTTP middleware that gates admin routes. It is the GDPR A0.1 layer.
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// PasswordCost is the bcrypt work factor. 12 takes ~0.3s on a modest Lambda
// and is well above current GPU brute-force comfort.
const PasswordCost = 12

// HashPassword returns a bcrypt hash of the plaintext password. Never log the
// returned string at info level — bcrypt hashes plus a known prefix can be
// fed to offline crackers.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("empty password")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), PasswordCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword returns nil on match. On mismatch it returns
// bcrypt.ErrMismatchedHashAndPassword (caller maps to 401). Constant-time
// inside bcrypt.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
