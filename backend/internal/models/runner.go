package models

import (
	"strings"
	"time"
)

type Runner struct {
	ID        string
	Name      string
	BirthDate string
	Gender    string
	CreatedAt time.Time
}

// NameDobKey returns the lookup key used by the byNameDOB GSI: the
// lower-cased, whitespace-trimmed name joined to the birth date.
func (r Runner) NameDobKey() string {
	return NameDobKey(r.Name, r.BirthDate)
}

func NameDobKey(name, birthDate string) string {
	return strings.ToLower(strings.TrimSpace(name)) + "#" + birthDate
}
