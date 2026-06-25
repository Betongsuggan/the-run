package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DSRSessionCookieName is the cookie that carries a /my-data session JWT.
// Distinct from SessionCookieName (admin) so the two can coexist on the same
// browser without interfering — an admin testing DSR on their own account
// has both cookies set.
const DSRSessionCookieName = "the-run-dsr-session"

// DSRSessionClaims is the payload of a DSR session JWT. No admin flag; the
// subject is always the account whose data this session is allowed to manage.
type DSRSessionClaims struct {
	Kind string `json:"k"` // always "dsr"; future-proofs against confused-deputy reuse
	jwt.RegisteredClaims
}

// DSRSessionTTL is how long a DSR session lives from issuance. Short — the
// magic-link flow re-issues cheaply, and the session grants full self-service
// edit + erase powers, so we want a tight window. The frontend can prompt
// re-auth without much UX cost.
const DSRSessionTTL = 2 * time.Hour

// IssueDSRSession returns a signed JWT for a /my-data session bound to the
// given account.
func IssueDSRSession(cfg Config, accountID string, now time.Time) (string, error) {
	if len(cfg.JWTSecret) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	claims := DSRSessionClaims{
		Kind: "dsr",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "the-run",
			Subject:   accountID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(DSRSessionTTL)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(cfg.JWTSecret)
}

// ParseDSRSession verifies a DSR session JWT and returns the parsed claims.
// Returns ErrSessionInvalid (wrapped with a reason) on any signature, expiry,
// or kind-mismatch failure. Mirrors ParseSession's interface so callers can
// treat them uniformly; the wrapped reason is logged by the middleware so
// production failures are diagnosable in CloudWatch.
func ParseDSRSession(cfg Config, token string) (*DSRSessionClaims, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: empty token", ErrSessionInvalid)
	}
	parsed, err := jwt.ParseWithClaims(token, &DSRSessionClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}
		return cfg.JWTSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrSessionInvalid, err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("%w: not valid", ErrSessionInvalid)
	}
	claims, ok := parsed.Claims.(*DSRSessionClaims)
	if !ok {
		return nil, fmt.Errorf("%w: claims type assertion failed", ErrSessionInvalid)
	}
	if claims.Kind != "dsr" {
		// Kind check prevents an admin session token from being interpreted
		// as a DSR session even if the claim shape matches enough to parse.
		return nil, fmt.Errorf("%w: wrong kind %q", ErrSessionInvalid, claims.Kind)
	}
	return claims, nil
}
