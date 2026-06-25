package models

import (
	"strings"
	"time"
)

// RunnerConsents groups per-runner consent decisions. Per-account decisions
// (e.g. marketing email) live on AccountConsents instead.
type RunnerConsents struct {
	PublicResults Consent
}

type Runner struct {
	ID        string
	AccountID string
	Name      string
	BirthDate string
	Gender    string
	Consents  RunnerConsents
	CreatedAt time.Time
	// DeletionPendingUntil is set when this runner is in the 30-day grace
	// window after the user requested erasure. Hidden from public + admin
	// reads; restorable until the retention job runs.
	DeletionPendingUntil *time.Time
}

// NameDobKey returns the lookup key used by the byNameDOB GSI: the
// lower-cased, whitespace-trimmed name joined to the birth date.
func (r Runner) NameDobKey() string {
	return NameDobKey(r.Name, r.BirthDate)
}

func NameDobKey(name, birthDate string) string {
	return strings.ToLower(strings.TrimSpace(name)) + "#" + birthDate
}
