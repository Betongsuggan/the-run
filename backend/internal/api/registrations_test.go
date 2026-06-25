package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
	"github.com/BirgerRydback/the-run/backend/internal/turnstile"
)

// fakeStore is a tiny in-memory implementation covering only the methods the
// registration handler touches. It embeds store.Store so any unimplemented
// method panics — tests that wander outside this surface fail loudly.
type fakeStore struct {
	store.Store

	accountsByEmail map[string]*models.Account
	runners         []models.Runner
	registrations   []models.Registration
}

func newFakeStore() *fakeStore {
	return &fakeStore{accountsByEmail: map[string]*models.Account{}}
}

func (f *fakeStore) GetAccountByEmail(_ context.Context, email string) (*models.Account, error) {
	if a, ok := f.accountsByEmail[email]; ok {
		clone := *a
		return &clone, nil
	}
	return nil, nil
}

func (f *fakeStore) CreateAccount(_ context.Context, a models.Account) error {
	if _, exists := f.accountsByEmail[a.Email]; exists {
		return store.ErrAlreadyExists
	}
	clone := a
	f.accountsByEmail[a.Email] = &clone
	return nil
}

func (f *fakeStore) RunnerByNameDOB(_ context.Context, key string) ([]models.Runner, error) {
	matches := []models.Runner{}
	for _, r := range f.runners {
		if r.NameDobKey() == key {
			matches = append(matches, r)
		}
	}
	return matches, nil
}

func (f *fakeStore) CreateRunner(_ context.Context, r models.Runner) error {
	f.runners = append(f.runners, r)
	return nil
}

func (f *fakeStore) CreateRegistration(_ context.Context, r models.Registration) error {
	for _, existing := range f.registrations {
		if existing.RaceID == r.RaceID && existing.RunnerID == r.RunnerID {
			return store.ErrAlreadyRegistered
		}
	}
	f.registrations = append(f.registrations, r)
	return nil
}

// buildTestAPI wires registerRegistrations onto a humatest API with a fixed
// PolicyVersion so tests can assert against it.
func buildTestAPI(t *testing.T, s store.Store) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "0.0.0"))
	// Skip Turnstile in tests; covered separately by the turnstile package's own tests.
	registerRegistrations(api, s, auth.Config{PrivacyVersion: "2026-08-01"}, turnstile.Config{SkipVerification: true})
	return api
}

func registerBody(email, name, dob string, publicResults, marketing bool) map[string]any {
	return map[string]any{
		"name":          name,
		"email":         email,
		"dateOfBirth":   dob,
		"gender":        "M",
		"raceId":        "race-1",
		"publicResults": publicResults,
		"marketing":     marketing,
	}
}

func decodeRegisterOutput(t *testing.T, body string) struct {
	ID       string `json:"id"`
	RunnerID string `json:"runnerId"`
	Status   string `json:"status"`
} {
	t.Helper()
	var out struct {
		ID       string `json:"id"`
		RunnerID string `json:"runnerId"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// First registration: new account + new runner. Both consents get stamped with
// the configured PolicyVersion and a recent timestamp.
func TestRegister_NewAccount_StampsBothConsents(t *testing.T) {
	s := newFakeStore()
	api := buildTestAPI(t, s)

	resp := api.Post("/registrations", registerBody("Runner@Example.com", "Anna Andersson", "1990-05-12", true, true))
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	if len(s.accountsByEmail) != 1 {
		t.Fatalf("want 1 account, got %d", len(s.accountsByEmail))
	}
	acc := s.accountsByEmail["runner@example.com"]
	if acc == nil {
		t.Fatalf("account not indexed by normalized email")
	}
	if !acc.Consents.Marketing.Granted {
		t.Errorf("marketing consent not granted")
	}
	if acc.Consents.Marketing.PolicyVersion != "2026-08-01" {
		t.Errorf("marketing policyVersion = %q, want %q", acc.Consents.Marketing.PolicyVersion, "2026-08-01")
	}
	if time.Since(acc.Consents.Marketing.At) > time.Minute {
		t.Errorf("marketing.At is not recent: %v", acc.Consents.Marketing.At)
	}

	if len(s.runners) != 1 {
		t.Fatalf("want 1 runner, got %d", len(s.runners))
	}
	r := s.runners[0]
	if !r.Consents.PublicResults.Granted {
		t.Errorf("publicResults consent not granted")
	}
	if r.Consents.PublicResults.PolicyVersion != "2026-08-01" {
		t.Errorf("publicResults policyVersion = %q, want %q", r.Consents.PublicResults.PolicyVersion, "2026-08-01")
	}
}

// Existing account, new runner under it: marketing must NOT be overwritten by
// the new form value (the first decision stands), but the new runner gets its
// own publicResults consent.
func TestRegister_ExistingAccount_LeavesMarketingUntouched(t *testing.T) {
	s := newFakeStore()
	api := buildTestAPI(t, s)

	// Seed: account already exists with marketing=false.
	priorAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.accountsByEmail["parent@example.com"] = &models.Account{
		ID:        "acct-1",
		Email:     "parent@example.com",
		CreatedAt: priorAt,
		Consents: models.AccountConsents{
			Marketing: models.Consent{Granted: false, At: priorAt, PolicyVersion: "2026-08-01"},
		},
	}

	// Parent registers a kid with marketing=TRUE in the form — must be ignored.
	resp := api.Post("/registrations", registerBody("parent@example.com", "Kid Andersson", "2015-04-01", false, true))
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	acc := s.accountsByEmail["parent@example.com"]
	if acc.Consents.Marketing.Granted {
		t.Errorf("marketing consent was overwritten to true; expected to remain false")
	}
	if !acc.Consents.Marketing.At.Equal(priorAt) {
		t.Errorf("marketing.At was overwritten: got %v, want %v", acc.Consents.Marketing.At, priorAt)
	}

	if len(s.runners) != 1 {
		t.Fatalf("want 1 runner, got %d", len(s.runners))
	}
	r := s.runners[0]
	if r.AccountID != "acct-1" {
		t.Errorf("new runner accountId = %q, want acct-1", r.AccountID)
	}
	if r.Consents.PublicResults.Granted {
		t.Errorf("publicResults should be false (opt-out form value)")
	}
	if r.Consents.PublicResults.PolicyVersion != "2026-08-01" {
		t.Errorf("publicResults policyVersion = %q, want %q", r.Consents.PublicResults.PolicyVersion, "2026-08-01")
	}
}

// Existing account + existing runner under that account: re-registration must
// not create duplicate Runner rows, and the prior runner's consent must remain
// unchanged regardless of the form value.
func TestRegister_ExistingRunner_PreservesConsent(t *testing.T) {
	s := newFakeStore()
	api := buildTestAPI(t, s)

	priorAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.accountsByEmail["self@example.com"] = &models.Account{
		ID:        "acct-2",
		Email:     "self@example.com",
		CreatedAt: priorAt,
	}
	s.runners = append(s.runners, models.Runner{
		ID:        "runner-1",
		AccountID: "acct-2",
		Name:      "Self Person",
		BirthDate: "1990-05-12",
		Consents: models.RunnerConsents{
			PublicResults: models.Consent{Granted: false, At: priorAt, PolicyVersion: "2026-08-01"},
		},
	})

	// Re-register with publicResults=true in the form — must be ignored, runner reused.
	resp := api.Post("/registrations", registerBody("self@example.com", "Self Person", "1990-05-12", true, false))
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	out := decodeRegisterOutput(t, resp.Body.String())
	if out.RunnerID != "runner-1" {
		t.Errorf("runnerId = %q, want runner-1 (existing runner should be reused)", out.RunnerID)
	}
	if len(s.runners) != 1 {
		t.Errorf("want 1 runner (reused), got %d", len(s.runners))
	}
	if s.runners[0].Consents.PublicResults.Granted {
		t.Errorf("publicResults was overwritten to true; expected to remain false")
	}
}
