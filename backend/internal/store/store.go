// Package store hides persistence behind a Store interface so handlers can be
// unit-tested against fakes and the DynamoDB implementation can evolve
// independently of the API surface.
package store

import (
	"context"
	"errors"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// ErrAlreadyRegistered is returned when a runner is already registered for a
// given race. The API layer maps this to HTTP 409.
var ErrAlreadyRegistered = errors.New("runner already registered for this race")

type Store interface {
	// RunnerByNameDOB returns the matching runner, or (nil, nil) on miss.
	RunnerByNameDOB(ctx context.Context, nameDobKey string) (*models.Runner, error)
	CreateRunner(ctx context.Context, r models.Runner) error
	// CreateRegistration returns ErrAlreadyRegistered if the same
	// (RaceID, RunnerID) pair already exists.
	CreateRegistration(ctx context.Context, reg models.Registration) error
}
