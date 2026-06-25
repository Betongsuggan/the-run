package api

import (
	"strings"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// redactRunnerForPublic blanks out fields in a public RunnerDTO based on the
// runner's consent and their age at the reference date. The caller decides
// what asOf means: the race's event date for result/leaderboard endpoints, or
// time.Now() for context-free runner listings.
//
// Rules (GDPR plan A0.8):
//   - Runner <13 at asOf → name becomes "First L.", birthYear and ID dropped,
//     regardless of consent. Defensible default for child data.
//   - Else if runner has opted out of public results → name becomes "Anonym",
//     ID dropped. BirthYear retained (it's coarse — just a year).
//   - Else → no change. Adult who's opted in.
//
// Returns true if any redaction happened, so the caller can clear adjacent
// fields (e.g. ResultDTO.RunnerID, which links to the now-hidden profile).
func redactRunnerForPublic(dto *RunnerDTO, runner models.Runner, asOf time.Time) bool {
	if isMinorAt(runner.BirthDate, asOf) {
		dto.Name = redactedMinorName(runner.Name)
		dto.BirthYear = nil
		dto.ID = ""
		return true
	}
	if !runner.Consents.PublicResults.Granted {
		dto.Name = "Anonym"
		dto.ID = ""
		return true
	}
	return false
}

// isMinorAt reports whether someone with the given YYYY-MM-DD birth date was
// under 13 on the reference date. Unparseable birth dates fall through as
// "not a minor" — the consent check still applies in that path.
func isMinorAt(birthDate string, asOf time.Time) bool {
	dob, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return false
	}
	age := asOf.Year() - dob.Year()
	if asOf.YearDay() < dob.YearDay() {
		age--
	}
	return age < 13
}

// redactedMinorName collapses "Anna Maria Andersson" to "Anna Maria A." —
// keeps any number of leading given names and reduces only the family name
// to its initial. Single-word names are returned unchanged (better than
// surfacing nothing).
func redactedMinorName(name string) string {
	parts := strings.Fields(name)
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	last := []rune(parts[len(parts)-1])
	if len(last) == 0 {
		return strings.Join(parts[:len(parts)-1], " ")
	}
	return strings.Join(parts[:len(parts)-1], " ") + " " + string(last[0]) + "."
}
