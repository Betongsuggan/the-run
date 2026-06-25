package models

import "time"

// Status values for a Registration. The public POST /registrations endpoint
// writes StatusReceived; the admin pipeline progresses entries through
// pending/finished/dnf/dns. Reads normalize "received" → "pending".
//
// StatusPendingGuardianConsent (GDPR A0.4) is set when a runner under 13 at
// the race date registers via a guardian: the registration is held until the
// guardian clicks the magic link emailed to them. Excluded from public
// leaderboards while in this state.
const (
	StatusReceived               = "received"
	StatusPending                = "pending"
	StatusPendingGuardianConsent = "pending_guardian_consent"
	StatusFinished               = "finished"
	StatusDNF                    = "dnf"
	StatusDNS                    = "dns"
)

type Category struct {
	Gender   string
	AgeGroup string
}

type Split struct {
	Km          int
	TimeSeconds int
}

type Registration struct {
	ID            string
	RaceID        string
	RunnerID      string
	Status        string
	Bib           string
	Category      *Category
	FinishSeconds *int
	Splits        []Split
	Conditions    string
	Notes         string
	CreatedAt     time.Time
}
