// Package store hides persistence behind a Store interface so handlers can be
// unit-tested against fakes and the DynamoDB implementation can evolve
// independently of the API surface.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// ErrAlreadyRegistered is returned when a runner is already registered for a
// given race. The API layer maps this to HTTP 409.
var ErrAlreadyRegistered = errors.New("runner already registered for this race")

// ErrNotFound is returned by Get/Update/Delete when the item does not exist.
// API layer maps this to HTTP 404.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned when a uniqueness constraint is violated on
// create (e.g. duplicate runner name+DOB). API layer maps this to HTTP 409.
var ErrAlreadyExists = errors.New("already exists")

// RegistrationUpdate carries the optional fields a bulk/single update can
// modify. nil fields are left unchanged; pointer-typed numeric fields use
// the zero pointer to mean "clear", as DynamoDB UpdateItem REMOVE.
type RegistrationUpdate struct {
	RaceID        string
	RunnerID      string
	Status        *string
	Bib           *string
	Category      *models.Category
	FinishSeconds *int
	ClearFinish   bool
	Splits        []models.Split
	ClearSplits   bool
	Conditions    *string
	Notes         *string
}

type Store interface {
	// Accounts
	// GetAccountByID returns ErrNotFound when no account with the given id exists.
	GetAccountByID(ctx context.Context, id string) (*models.Account, error)
	// GetAccountByEmail returns (nil, nil) when no account is bound to the email
	// (so callers can use the find-or-create pattern without distinguishing
	// "missing" from "real error"). Returns ErrNotFound only when the email
	// sentinel exists but the primary row is missing — an inconsistency.
	GetAccountByEmail(ctx context.Context, email string) (*models.Account, error)
	CreateAccount(ctx context.Context, a models.Account) error
	UpdateAccount(ctx context.Context, a models.Account) error
	TouchAccountLastLogin(ctx context.Context, id string, at time.Time) error

	// Auth attempts — failed-login tracking with TTL, drives lockout.
	RecordAuthAttempt(ctx context.Context, accountID string, at time.Time, ttl time.Duration) error
	CountActiveAuthAttempts(ctx context.Context, accountID string, now time.Time) (int, error)
	ClearAuthAttempts(ctx context.Context, accountID string) error

	// Runners
	// RunnerByNameDOB may return multiple Runner records — one per Account
	// that has someone with this (name, DOB). Callers filter by AccountID.
	RunnerByNameDOB(ctx context.Context, nameDobKey string) ([]models.Runner, error)
	ListRunners(ctx context.Context) ([]models.Runner, error)
	ListRunnersByAccount(ctx context.Context, accountID string) ([]models.Runner, error)
	GetRunner(ctx context.Context, id string) (*models.Runner, error)
	CreateRunner(ctx context.Context, r models.Runner) error
	UpdateRunner(ctx context.Context, r models.Runner) error
	DeleteRunner(ctx context.Context, id string) error

	// Events
	ListEvents(ctx context.Context) ([]models.Event, error)
	GetEvent(ctx context.Context, id string) (*models.Event, error)
	CreateEvent(ctx context.Context, e models.Event) error
	UpdateEvent(ctx context.Context, e models.Event) error
	DeleteEvent(ctx context.Context, id string) error

	// Races
	ListRaces(ctx context.Context) ([]models.Race, error)
	ListRacesByEvent(ctx context.Context, eventID string) ([]models.Race, error)
	GetRace(ctx context.Context, id string) (*models.Race, error)
	CreateRace(ctx context.Context, r models.Race) error
	UpdateRace(ctx context.Context, r models.Race) error
	DeleteRace(ctx context.Context, id string) error

	// Registrations
	// CreateRegistration returns ErrAlreadyRegistered if the same
	// (RaceID, RunnerID) pair already exists.
	CreateRegistration(ctx context.Context, reg models.Registration) error
	ListRegistrations(ctx context.Context) ([]models.Registration, error)
	ListRegistrationsByRace(ctx context.Context, raceID string) ([]models.Registration, error)
	ListRegistrationsByRunner(ctx context.Context, runnerID string) ([]models.Registration, error)
	GetRegistration(ctx context.Context, raceID, runnerID string) (*models.Registration, error)
	GetRegistrationByID(ctx context.Context, id string) (*models.Registration, error)
	UpdateRegistration(ctx context.Context, u RegistrationUpdate) (*models.Registration, error)
	DeleteRegistration(ctx context.Context, raceID, runnerID string) error
}
