package models

import "time"

type Registration struct {
	ID        string
	RaceID    string
	RunnerID  string
	Status    string
	CreatedAt time.Time
}
