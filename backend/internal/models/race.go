package models

import "time"

type Race struct {
	ID             string
	EventID        string
	Name           string
	DistanceMeters int
	Discipline     string
	// MaxRunners caps how many runners can self-register via the public
	// form. Zero means unlimited. The admin path bypasses this check.
	MaxRunners int
	CreatedAt  time.Time
}
