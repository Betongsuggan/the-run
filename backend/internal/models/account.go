package models

import (
	"strings"
	"time"
)

// Consent records a single granted/withdrawn decision against a versioned
// privacy policy. Used by both Account (marketing) and Runner (public results).
//
// PolicyID + PolicyRevision pin the exact policy revision the user accepted —
// the body of that revision is preserved in the policy-revisions table so the
// runner can always view the text they consented to. PolicyVersion is the
// admin-chosen slug (e.g. "2026-08-01") kept alongside for display without
// needing a lookup.
type Consent struct {
	Granted        bool
	At             time.Time
	PolicyID       string
	PolicyRevision int
	PolicyVersion  string
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
	// DeletionPendingUntil is set when the user requested account erasure;
	// the retention job hard-purges PII after this time. Until then the
	// account is hidden from all reads except DSR (so the user can still
	// restore) and the retention job.
	DeletionPendingUntil *time.Time
}

// NormalizeEmail lowercases and trims an email so two registrations writing
// the same address (modulo case/whitespace) collide on the email sentinel.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
