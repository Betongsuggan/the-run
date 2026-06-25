package models

import "time"

// Status values for a Registration. The public POST /registrations endpoint
// writes StatusReceived; the admin pipeline progresses entries through
// pending/finished/dnf/dns. Reads normalize "received" → "pending".
const (
	StatusReceived = "received"
	StatusPending  = "pending"
	StatusFinished = "finished"
	StatusDNF      = "dnf"
	StatusDNS      = "dns"
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
