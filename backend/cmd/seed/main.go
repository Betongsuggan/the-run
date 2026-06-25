// Command seed pushes the dev fixture (mirrors frontend/src/lib/mock/data.ts)
// into a running API. Idempotent on re-run: existing runners/events/races are
// reused by natural key (runner = name+birthYear, event = name+year, race =
// name+eventId), and existing registrations return 409 which we ignore.
//
// Usage:
//
//	API_BASE_URL=http://localhost:8080 go run ./cmd/seed
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const defaultAPIBase = "http://localhost:8080"

type fixtureRunner struct {
	Key       string `json:"-"`
	Name      string `json:"name"`
	Gender    string `json:"gender"`
	BirthYear int    `json:"birthYear"`
}

type fixtureEvent struct {
	Key      string `json:"-"`
	Name     string `json:"name"`
	Year     int    `json:"year"`
	Date     string `json:"date"`
	Location string `json:"location"`
}

type fixtureRace struct {
	Key            string `json:"-"`
	EventKey       string `json:"-"`
	Name           string `json:"name"`
	DistanceMeters int    `json:"distanceMeters"`
	Discipline     string `json:"discipline"`
}

type fixtureSplit struct {
	Km          int `json:"km"`
	TimeSeconds int `json:"timeSeconds"`
}

type fixtureCategory struct {
	Gender   string `json:"gender,omitempty"`
	AgeGroup string `json:"ageGroup,omitempty"`
}

type fixtureRegistration struct {
	RunnerKey     string           `json:"-"`
	RaceKey       string           `json:"-"`
	Bib           string           `json:"bib,omitempty"`
	Category      *fixtureCategory `json:"category,omitempty"`
	Status        string           `json:"status"`
	FinishSeconds *int             `json:"finishSeconds,omitempty"`
	Splits        []fixtureSplit   `json:"splits,omitempty"`
	Conditions    string           `json:"conditions,omitempty"`
	Notes         string           `json:"notes,omitempty"`
}

func splits5K(total int) []fixtureSplit {
	per := total / 5
	out := make([]fixtureSplit, 5)
	for i := range 5 {
		out[i] = fixtureSplit{Km: i + 1, TimeSeconds: per + (i-2)*3}
	}
	return out
}

func splits10K(total int) []fixtureSplit {
	per := total / 10
	out := make([]fixtureSplit, 10)
	for i := range 10 {
		out[i] = fixtureSplit{Km: i + 1, TimeSeconds: per + (i-5)*4}
	}
	return out
}

func main() {
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	c := &client{base: strings.TrimRight(apiBase, "/")}

	runners := []fixtureRunner{
		{Key: "birger", Name: "Birger Rydback", Gender: "M", BirthYear: 1988},
		{Key: "luisa", Name: "Luisa Stock", Gender: "F", BirthYear: 1991},
		{Key: "kalle", Name: "Kalle Torlén", Gender: "M", BirthYear: 1975},
		{Key: "viktoria", Name: "Viktoria Torlén", Gender: "F", BirthYear: 2014},
	}

	events := []fixtureEvent{
		{Key: "2025", Name: "Ingmarsöloppet 2025", Year: 2025, Date: "2025-06-14", Location: "Stockholm"},
		{Key: "2026", Name: "Ingmarsöloppet 2026", Year: 2026, Date: "2026-06-13", Location: "Stockholm"},
	}

	races := []fixtureRace{
		{Key: "2025-5k-run", EventKey: "2025", Name: "5K Run", DistanceMeters: 5000, Discipline: "run"},
		{Key: "2025-5k-walk", EventKey: "2025", Name: "5K Walk", DistanceMeters: 5000, Discipline: "walk"},
		{Key: "2025-10k-run", EventKey: "2025", Name: "10K Run", DistanceMeters: 10000, Discipline: "run"},
		{Key: "2025-kids", EventKey: "2025", Name: "Kids' Race", DistanceMeters: 1000, Discipline: "kids"},
		{Key: "2026-5k-run", EventKey: "2026", Name: "5K Run", DistanceMeters: 5000, Discipline: "run"},
		{Key: "2026-5k-walk", EventKey: "2026", Name: "5K Walk", DistanceMeters: 5000, Discipline: "walk"},
		{Key: "2026-10k-run", EventKey: "2026", Name: "10K Run", DistanceMeters: 10000, Discipline: "run"},
		{Key: "2026-kids", EventKey: "2026", Name: "Kids' Race", DistanceMeters: 1000, Discipline: "kids"},
	}

	intp := func(v int) *int { return &v }

	registrations := []fixtureRegistration{
		{RunnerKey: "birger", RaceKey: "2025-5k-walk", Bib: "142", Category: &fixtureCategory{Gender: "M", AgeGroup: "M30-39"}, Status: "finished", FinishSeconds: intp(36*60 + 42), Splits: splits5K(36*60 + 42), Conditions: "Sunny, 18°C, light wind", Notes: "Nice steady tempo all the way."},
		{RunnerKey: "birger", RaceKey: "2026-5k-walk", Bib: "88", Category: &fixtureCategory{Gender: "M", AgeGroup: "M30-39"}, Status: "finished", FinishSeconds: intp(35*60 + 18), Splits: splits5K(35*60 + 18), Conditions: "Overcast, 14°C", Notes: "New 5K walk PB."},
		{RunnerKey: "luisa", RaceKey: "2026-10k-run", Bib: "301", Category: &fixtureCategory{Gender: "F", AgeGroup: "F30-39"}, Status: "finished", FinishSeconds: intp(53*60 + 24), Splits: splits10K(53*60 + 24), Conditions: "Overcast, 15°C", Notes: "First 10K. Held back the first half and finished strong."},
		{RunnerKey: "kalle", RaceKey: "2025-10k-run", Bib: "210", Category: &fixtureCategory{Gender: "M", AgeGroup: "M50-59"}, Status: "finished", FinishSeconds: intp(45*60 + 12), Splits: splits10K(45*60 + 12), Conditions: "Sunny, 18°C", Notes: "Veteran category win."},
		{RunnerKey: "kalle", RaceKey: "2026-10k-run", Bib: "210", Category: &fixtureCategory{Gender: "M", AgeGroup: "M50-59"}, Status: "finished", FinishSeconds: intp(44*60 + 58), Conditions: "Overcast, 15°C"},
		{RunnerKey: "viktoria", RaceKey: "2025-kids", Bib: "K-44", Category: &fixtureCategory{Gender: "F", AgeGroup: "U12"}, Status: "finished", FinishSeconds: intp(6*60 + 12), Conditions: "Sunny, 18°C", Notes: "First medal!"},
		{RunnerKey: "viktoria", RaceKey: "2026-kids", Bib: "K-44", Category: &fixtureCategory{Gender: "F", AgeGroup: "U12"}, Status: "finished", FinishSeconds: intp(5*60 + 48), Conditions: "Overcast, 15°C"},
	}

	log.Printf("seeding against %s", c.base)

	runnerIDs, err := upsertRunners(c, runners)
	if err != nil {
		log.Fatalf("seed runners: %v", err)
	}
	eventIDs, err := upsertEvents(c, events)
	if err != nil {
		log.Fatalf("seed events: %v", err)
	}
	raceIDs, err := upsertRaces(c, races, eventIDs)
	if err != nil {
		log.Fatalf("seed races: %v", err)
	}
	if err := upsertRegistrations(c, registrations, runnerIDs, raceIDs); err != nil {
		log.Fatalf("seed registrations: %v", err)
	}

	log.Printf("seed complete: %d runners, %d events, %d races, %d registrations",
		len(runnerIDs), len(eventIDs), len(raceIDs), len(registrations))
}

// ─── client ────────────────────────────────────────────────────────────

type client struct {
	base string
}

type apiError struct {
	Status int
	Body   string
}

func (e *apiError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body) }

func (c *client) do(method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return out, &apiError{Status: res.StatusCode, Body: string(out)}
	}
	return out, nil
}

// ─── upsert helpers ────────────────────────────────────────────────────

type runnerResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BirthYear *int   `json:"birthYear,omitempty"`
}

func upsertRunners(c *client, runners []fixtureRunner) (map[string]string, error) {
	existing, err := c.do("GET", "/runners", nil)
	if err != nil {
		return nil, err
	}
	var list struct {
		Runners []runnerResponse `json:"runners"`
	}
	if err := json.Unmarshal(existing, &list); err != nil {
		return nil, err
	}
	bynk := map[string]string{}
	for _, r := range list.Runners {
		yr := 0
		if r.BirthYear != nil {
			yr = *r.BirthYear
		}
		bynk[runnerKey(r.Name, yr)] = r.ID
	}

	ids := map[string]string{}
	for _, r := range runners {
		if id, ok := bynk[runnerKey(r.Name, r.BirthYear)]; ok {
			ids[r.Key] = id
			continue
		}
		body := map[string]any{
			"name":      r.Name,
			"gender":    r.Gender,
			"birthYear": r.BirthYear,
		}
		out, err := c.do("POST", "/runners", body)
		if err != nil {
			return nil, fmt.Errorf("create runner %q: %w", r.Name, err)
		}
		var created runnerResponse
		if err := json.Unmarshal(out, &created); err != nil {
			return nil, err
		}
		ids[r.Key] = created.ID
	}
	return ids, nil
}

type eventResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

func upsertEvents(c *client, events []fixtureEvent) (map[string]string, error) {
	existing, err := c.do("GET", "/events", nil)
	if err != nil {
		return nil, err
	}
	var list struct {
		Events []eventResponse `json:"events"`
	}
	if err := json.Unmarshal(existing, &list); err != nil {
		return nil, err
	}
	bynk := map[string]string{}
	for _, e := range list.Events {
		bynk[eventKey(e.Name, e.Year)] = e.ID
	}

	ids := map[string]string{}
	for _, e := range events {
		if id, ok := bynk[eventKey(e.Name, e.Year)]; ok {
			ids[e.Key] = id
			continue
		}
		body := map[string]any{
			"name":     e.Name,
			"year":     e.Year,
			"date":     e.Date,
			"location": e.Location,
		}
		out, err := c.do("POST", "/events", body)
		if err != nil {
			return nil, fmt.Errorf("create event %q: %w", e.Name, err)
		}
		var created eventResponse
		if err := json.Unmarshal(out, &created); err != nil {
			return nil, err
		}
		ids[e.Key] = created.ID
	}
	return ids, nil
}

type raceResponse struct {
	ID      string `json:"id"`
	EventID string `json:"eventId"`
	Name    string `json:"name"`
}

func upsertRaces(c *client, races []fixtureRace, eventIDs map[string]string) (map[string]string, error) {
	ids := map[string]string{}
	for eventKey, eventID := range eventIDs {
		existing, err := c.do("GET", "/events/"+url.PathEscape(eventID)+"/races", nil)
		if err != nil {
			return nil, err
		}
		var list struct {
			Races []raceResponse `json:"races"`
		}
		if err := json.Unmarshal(existing, &list); err != nil {
			return nil, err
		}
		byName := map[string]string{}
		for _, r := range list.Races {
			byName[r.Name] = r.ID
		}
		for _, r := range races {
			if r.EventKey != eventKey {
				continue
			}
			if id, ok := byName[r.Name]; ok {
				ids[r.Key] = id
				continue
			}
			body := map[string]any{
				"eventId":        eventID,
				"name":           r.Name,
				"distanceMeters": r.DistanceMeters,
				"discipline":     r.Discipline,
			}
			out, err := c.do("POST", "/races", body)
			if err != nil {
				return nil, fmt.Errorf("create race %q: %w", r.Name, err)
			}
			var created raceResponse
			if err := json.Unmarshal(out, &created); err != nil {
				return nil, err
			}
			ids[r.Key] = created.ID
		}
	}
	return ids, nil
}

func upsertRegistrations(c *client, regs []fixtureRegistration, runnerIDs, raceIDs map[string]string) error {
	for _, reg := range regs {
		runnerID, ok := runnerIDs[reg.RunnerKey]
		if !ok {
			return fmt.Errorf("registration references unknown runner %q", reg.RunnerKey)
		}
		raceID, ok := raceIDs[reg.RaceKey]
		if !ok {
			return fmt.Errorf("registration references unknown race %q", reg.RaceKey)
		}
		body := map[string]any{
			"raceId":   raceID,
			"runnerId": runnerID,
		}
		if reg.Bib != "" {
			body["bib"] = reg.Bib
		}
		if reg.Category != nil {
			body["category"] = reg.Category
		}
		out, err := c.do("POST", "/registrations/admin", body)
		var apiErr *apiError
		switch {
		case err == nil:
			// continue to PATCH to fill in finish data
		case isAPIErrorWithStatus(err, &apiErr, http.StatusConflict):
			// already exists — PATCH it instead.
		default:
			return fmt.Errorf("create registration %s/%s: %w", reg.RunnerKey, reg.RaceKey, err)
		}
		_ = out

		patch := map[string]any{
			"status": reg.Status,
		}
		if reg.Bib != "" {
			patch["bib"] = reg.Bib
		}
		if reg.Category != nil {
			patch["category"] = reg.Category
		}
		if reg.FinishSeconds != nil {
			patch["finishSeconds"] = *reg.FinishSeconds
		}
		if len(reg.Splits) > 0 {
			patch["splits"] = reg.Splits
		}
		if reg.Conditions != "" {
			patch["conditions"] = reg.Conditions
		}
		if reg.Notes != "" {
			patch["notes"] = reg.Notes
		}
		path := "/races/" + url.PathEscape(raceID) + "/registrations/" + url.PathEscape(runnerID)
		if _, err := c.do("PATCH", path, patch); err != nil {
			return fmt.Errorf("patch registration %s/%s: %w", reg.RunnerKey, reg.RaceKey, err)
		}
	}
	return nil
}

func runnerKey(name string, year int) string {
	return strings.ToLower(strings.TrimSpace(name)) + "#" + fmt.Sprintf("%d", year)
}

func eventKey(name string, year int) string {
	return strings.ToLower(strings.TrimSpace(name)) + "#" + fmt.Sprintf("%d", year)
}

func isAPIErrorWithStatus(err error, target **apiError, status int) bool {
	var apiErr *apiError
	if !errAsAPI(err, &apiErr) {
		return false
	}
	if apiErr.Status != status {
		return false
	}
	*target = apiErr
	return true
}

func errAsAPI(err error, target **apiError) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*apiError); ok {
		*target = apiErr
		return true
	}
	return false
}
