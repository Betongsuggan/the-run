package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionCookieName is the cookie that carries the JWT post-MFA. HttpOnly,
// Secure, SameSite=Strict.
const SessionCookieName = "the-run-session"

// SessionClaims is the payload of the post-MFA session JWT.
type SessionClaims struct {
	IsAdmin bool `json:"adm"`
	jwt.RegisteredClaims
}

// IssueSession returns a signed JWT for the given account. Use BindSessionCookie
// to attach it to an HTTP response.
func IssueSession(cfg Config, accountID string, isAdmin bool, now time.Time) (string, error) {
	if len(cfg.JWTSecret) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	claims := SessionClaims{
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "the-run",
			Subject:   accountID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.SessionTTL) * time.Second)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(cfg.JWTSecret)
}

// ParseSession verifies a session JWT against the configured secret and
// returns the parsed claims. Returns ErrSessionInvalid on any signature or
// expiry failure (callers don't need to distinguish).
var ErrSessionInvalid = errors.New("session invalid")

func ParseSession(cfg Config, token string) (*SessionClaims, error) {
	if token == "" {
		return nil, ErrSessionInvalid
	}
	parsed, err := jwt.ParseWithClaims(token, &SessionClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}
		return cfg.JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrSessionInvalid
	}
	claims, ok := parsed.Claims.(*SessionClaims)
	if !ok {
		return nil, ErrSessionInvalid
	}
	return claims, nil
}

// BindSessionCookie attaches the session JWT as a hardened cookie. SameSite=Strict
// is safe for us because frontend and API share the registrable domain
// (ingmarsoloppet.se → run.ingmarsoloppet.se & api.run.ingmarsoloppet.se).
func BindSessionCookie(w http.ResponseWriter, cfg Config, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   cfg.CookieDomain,
		MaxAge:   cfg.SessionTTL,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearSessionCookie sets an expired cookie so the browser drops the session.
func ClearSessionCookie(w http.ResponseWriter, cfg Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

// ReadSessionCookie pulls the JWT from the request without verifying it.
func ReadSessionCookie(r *http.Request) string {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c == nil {
		return ""
	}
	return c.Value
}
