package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ChallengeCookieName carries the short-lived state between the password and
// TOTP steps. Cleared once the TOTP verification succeeds.
const ChallengeCookieName = "the-run-auth-challenge"

// ChallengeStep distinguishes "user is enrolling their first authenticator"
// from "user is verifying an existing one".
type ChallengeStep string

const (
	StepVerify ChallengeStep = "verify"
	StepEnroll ChallengeStep = "enroll"
)

// ChallengeClaims is signed with the same JWT secret as the session token.
type ChallengeClaims struct {
	Step         ChallengeStep `json:"step"`
	EnrollSecret string        `json:"enrollSecret,omitempty"`
	jwt.RegisteredClaims
}

// IssueChallenge signs a short-lived JWT containing the account id and the
// next step. For enrollment, the TOTP secret travels in the cookie so the
// server is stateless between login and totp.
func IssueChallenge(cfg Config, accountID string, step ChallengeStep, enrollSecret string, now time.Time) (string, error) {
	if len(cfg.JWTSecret) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	claims := ChallengeClaims{
		Step:         step,
		EnrollSecret: enrollSecret,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "the-run",
			Subject:   accountID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.ChallengeTTL) * time.Second)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(cfg.JWTSecret)
}

// ErrChallengeInvalid covers any failure to parse or verify the challenge.
var ErrChallengeInvalid = errors.New("challenge invalid")

func ParseChallenge(cfg Config, token string) (*ChallengeClaims, error) {
	if token == "" {
		return nil, ErrChallengeInvalid
	}
	parsed, err := jwt.ParseWithClaims(token, &ChallengeClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return cfg.JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrChallengeInvalid
	}
	claims, ok := parsed.Claims.(*ChallengeClaims)
	if !ok {
		return nil, ErrChallengeInvalid
	}
	return claims, nil
}

func BindChallengeCookie(w http.ResponseWriter, cfg Config, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     ChallengeCookieName,
		Value:    token,
		Path:     "/",
		Domain:   cfg.CookieDomain,
		MaxAge:   cfg.ChallengeTTL,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearChallengeCookie(w http.ResponseWriter, cfg Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     ChallengeCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func ReadChallengeCookie(r *http.Request) string {
	c, err := r.Cookie(ChallengeCookieName)
	if err != nil || c == nil {
		return ""
	}
	return c.Value
}
