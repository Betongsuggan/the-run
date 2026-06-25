package models

import "time"

type Event struct {
	ID        string
	Name      string
	Year      int
	Date      string
	Location  string
	CreatedAt time.Time
}
