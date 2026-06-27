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
	StatusPendingDeletion        = "pending_deletion"
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
	RunnerID      string `gdpr:"identifying;purposes=registration,public-results"`
	Status        string
	Bib           string `gdpr:"behavioural;purposes=registration,public-results"`
	Category      *Category
	FinishSeconds *int    `gdpr:"behavioural;purposes=public-results"`
	Splits        []Split `gdpr:"behavioural;purposes=public-results"`
	Conditions    string
	Notes         string
	// PaymentReceivedAt is set when an admin (or, in a future iteration, a
	// Swish webhook) confirms the registration fee has been paid. Nil means
	// unpaid. For races with Race.RegistrationFeeOre == 0 this stays nil —
	// IsCleared treats the runner as paid implicitly.
	PaymentReceivedAt *time.Time
	CreatedAt         time.Time
}

// IsCleared reports whether the runner has satisfied the payment requirement
// for the given race. Free races are always cleared; paid races require
// PaymentReceivedAt to be set.
func (r Registration) IsCleared(race Race) bool {
	return race.RegistrationFeeOre == 0 || r.PaymentReceivedAt != nil
}
