package api

import (
	"context"
	"testing"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

func consentGranted(granted bool) models.RunnerConsents {
	return models.RunnerConsents{
		PublicResults: models.Consent{Granted: granted, At: time.Now(), PolicyVersion: "2026-08-01"},
	}
}

func TestRedactRunnerForPublic_AdultWithConsent_NoChange(t *testing.T) {
	r := models.Runner{
		ID:        "r-1",
		Name:      "Anna Andersson",
		BirthDate: "1990-05-12",
		Gender:    "F",
		Consents:  consentGranted(true),
	}
	dto := RunnerDTO{ID: r.ID, Name: r.Name, Gender: r.Gender, BirthYear: birthYearFromDate(r.BirthDate)}
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	changed := redactRunnerForPublic(&dto, r, asOf)
	if changed {
		t.Fatalf("did not expect redaction for consenting adult")
	}
	if dto.Name != "Anna Andersson" || dto.ID != "r-1" {
		t.Errorf("dto mutated: %+v", dto)
	}
}

func TestRedactRunnerForPublic_AdultOptedOut_Anonymized(t *testing.T) {
	r := models.Runner{
		ID:        "r-1",
		Name:      "Anna Andersson",
		BirthDate: "1990-05-12",
		Consents:  consentGranted(false),
	}
	dto := RunnerDTO{ID: r.ID, Name: r.Name, Gender: r.Gender, BirthYear: birthYearFromDate(r.BirthDate)}
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	if !redactRunnerForPublic(&dto, r, asOf) {
		t.Fatalf("expected redaction signal")
	}
	if dto.Name != "Anonym" {
		t.Errorf("name = %q, want Anonym", dto.Name)
	}
	if dto.ID != "" {
		t.Errorf("id should be cleared, got %q", dto.ID)
	}
	if dto.BirthYear == nil || *dto.BirthYear != 1990 {
		t.Errorf("birthYear should be retained for opted-out adult, got %v", dto.BirthYear)
	}
}

func TestRedactRunnerForPublic_Minor_RedactsRegardlessOfConsent(t *testing.T) {
	r := models.Runner{
		ID:        "kid-1",
		Name:      "Kalle Karlsson",
		BirthDate: "2015-04-01",
		// Even with publicResults granted, minor rule wins.
		Consents: consentGranted(true),
	}
	dto := RunnerDTO{ID: r.ID, Name: r.Name, Gender: r.Gender, BirthYear: birthYearFromDate(r.BirthDate)}
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // 11 years old.

	if !redactRunnerForPublic(&dto, r, asOf) {
		t.Fatalf("expected redaction signal")
	}
	if dto.Name != "Kalle K." {
		t.Errorf("name = %q, want %q", dto.Name, "Kalle K.")
	}
	if dto.ID != "" {
		t.Errorf("id should be cleared for minor, got %q", dto.ID)
	}
	if dto.BirthYear != nil {
		t.Errorf("birthYear should be dropped for minor, got %v", *dto.BirthYear)
	}
}

// Boundary: 13 years and 1 day old → adult rule, consent check applies.
func TestRedactRunnerForPublic_AgeBoundary_13YearsOneDay(t *testing.T) {
	r := models.Runner{
		ID:        "teen-1",
		Name:      "Teen Tester",
		BirthDate: "2013-06-30",
		Consents:  consentGranted(true),
	}
	dto := RunnerDTO{ID: r.ID, Name: r.Name, Gender: r.Gender, BirthYear: birthYearFromDate(r.BirthDate)}
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // 13 years + 1 day.

	if redactRunnerForPublic(&dto, r, asOf) {
		t.Errorf("13-year-old with consent should not be redacted")
	}
	if dto.Name != "Teen Tester" {
		t.Errorf("name unexpectedly changed to %q", dto.Name)
	}
}

func TestIsMinorAt_Boundaries(t *testing.T) {
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		dob  string
		want bool
	}{
		{"12y", "2014-07-01", true},
		{"day-before-13th-birthday", "2013-07-02", true},
		{"exactly-13", "2013-07-01", false},
		{"day-after-13th-birthday", "2013-06-30", false},
		{"adult", "1990-01-01", false},
		{"empty-birthdate", "", false},
		{"garbage-birthdate", "not-a-date", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isMinorAt(c.dob, asOf); got != c.want {
				t.Errorf("isMinorAt(%q) = %v, want %v", c.dob, got, c.want)
			}
		})
	}
}

// expandResult must clear the sibling ResultDTO.RunnerID when the runner's
// profile link is hidden, so the frontend can't reconstruct a navigation path
// the redaction was meant to deny.
func TestExpandResult_RedactsResultRunnerID(t *testing.T) {
	runner := models.Runner{
		ID:        "r-1",
		Name:      "Anna Andersson",
		BirthDate: "1990-05-12",
		Consents:  consentGranted(false),
	}
	rc := raceContext{
		race:  models.Race{ID: "race-1", EventID: "evt-1"},
		event: models.Event{ID: "evt-1", Date: "2026-07-01"},
		runner: func(_ string) (*models.Runner, error) {
			return &runner, nil
		},
	}
	fs1 := 1800
	res := ResultDTO{ID: "reg-1", RaceID: "race-1", RunnerID: "r-1", Bib: "42", FinishSeconds: &fs1}

	out, err := expandResult(context.Background(), res, rc)
	if err != nil {
		t.Fatalf("expandResult: %v", err)
	}
	if out.Runner.Name != "Anonym" {
		t.Errorf("runner.name = %q, want Anonym", out.Runner.Name)
	}
	if out.Runner.ID != "" || out.RunnerID != "" {
		t.Errorf("runner ID not cleared: runnerDTO.id=%q result.runnerId=%q", out.Runner.ID, out.RunnerID)
	}
	if out.Bib != "42" || out.FinishSeconds == nil || *out.FinishSeconds != 1800 {
		t.Errorf("leaderboard fields lost in redaction: %+v", out)
	}
}

// Minor in a race they ran while <13 stays redacted even when they're 13+ now.
func TestExpandResult_MinorAgeFrozenAtRaceDate(t *testing.T) {
	runner := models.Runner{
		ID:        "kid-1",
		Name:      "Kalle Karlsson",
		BirthDate: "2015-04-01",
		Consents:  consentGranted(true),
	}
	rc := raceContext{
		race:   models.Race{ID: "race-2", EventID: "evt-2"},
		event:  models.Event{ID: "evt-2", Date: "2026-07-01"}, // kid is 11.
		runner: func(_ string) (*models.Runner, error) { return &runner, nil },
	}
	fs2 := 600
	res := ResultDTO{ID: "reg-2", RaceID: "race-2", RunnerID: "kid-1", FinishSeconds: &fs2}

	out, err := expandResult(context.Background(), res, rc)
	if err != nil {
		t.Fatalf("expandResult: %v", err)
	}
	if out.Runner.Name != "Kalle K." {
		t.Errorf("runner.name = %q, want Kalle K.", out.Runner.Name)
	}
	if out.Runner.BirthYear != nil {
		t.Errorf("birthYear should be dropped for minor, got %v", *out.Runner.BirthYear)
	}
}

func TestRedactedMinorName_NameShapes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Anna Andersson", "Anna A."},
		{"Anna Maria Andersson", "Anna Maria A."},
		{"Cher", "Cher"},
		{"", ""},
		{"  Anna   Andersson  ", "Anna A."},
		{"Östen Ödegård", "Östen Ö."}, // Non-ASCII initial.
	}
	for _, c := range cases {
		if got := redactedMinorName(c.in); got != c.want {
			t.Errorf("redactedMinorName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
