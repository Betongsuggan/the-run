package models

import "time"

type Race struct {
	ID             string
	EventID        string
	Name           string
	DistanceMeters int
	Discipline     string
	CreatedAt      time.Time
}
