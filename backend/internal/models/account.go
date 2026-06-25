package models

import (
	"strings"
	"time"
)

// Consent records a single granted/withdrawn decision against a versioned
// privacy policy. Used by both Account (marketing) and Runner (public results).
type Consent struct {
	Granted       bool
	At            time.Time
	PolicyVersion string
}

// AccountConsents groups the consent decisions that live on the Account.
// Per-runner consents (e.g. public results) live on the Runner record.
type AccountConsents struct {
	Marketing Consent
}

// Account is the per-email identity record. One account can own many Runners
// (a parent who registers themselves + their kids). Admin accounts additionally
// carry PasswordHash and MFASecret; runner-only accounts leave them empty.
type Account struct {
	ID           string
	Email        string
	PasswordHash string
	MFASecret    string
	IsAdmin      bool
	Consents     AccountConsents
	Locale       string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// NormalizeEmail lowercases and trims an email so two registrations writing
// the same address (modulo case/whitespace) collide on the email sentinel.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
