package auth

import (
	"context"
	"errors"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// ErrLockedOut is returned when an account has exceeded MaxAttempts unsuccessful
// logins in the current AttemptTTL window. Callers should map to HTTP 429.
var ErrLockedOut = errors.New("account temporarily locked due to repeated failures")

// EnsureNotLocked refuses the login flow when the account is currently locked.
// Call BEFORE verifying the password — otherwise an attacker can probe whether
// a password is correct while locked out (a timing side channel).
func EnsureNotLocked(ctx context.Context, s store.Store, cfg Config, accountID string, now time.Time) error {
	count, err := s.CountActiveAuthAttempts(ctx, accountID, now)
	if err != nil {
		return err
	}
	if count >= cfg.MaxAttempts {
		return ErrLockedOut
	}
	return nil
}

// RecordFailedAttempt writes a TTL'd row in the auth-attempts table. The TTL
// matches the lockout window, so old failures fall off automatically.
func RecordFailedAttempt(ctx context.Context, s store.Store, cfg Config, accountID string, now time.Time) error {
	return s.RecordAuthAttempt(ctx, accountID, now, time.Duration(cfg.AttemptTTL)*time.Second)
}

// ClearAttempts wipes the failure record on a successful login.
func ClearAttempts(ctx context.Context, s store.Store, accountID string) error {
	return s.ClearAuthAttempts(ctx, accountID)
}
