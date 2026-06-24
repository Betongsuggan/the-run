//go:build integration

package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/google/uuid"
)

// Exercises the DynamoDB store against a running LocalStack with the tables
// already provisioned (e.g. via `just localstack-deploy`).
//
// Run with:
//
//	AWS_ENDPOINT_URL=http://localhost:4566 \
//	AWS_REGION=eu-north-1 \
//	AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
//	RUNNERS_TABLE_NAME=the-run-runners \
//	REGISTRATIONS_TABLE_NAME=the-run-registrations \
//	go test -tags=integration ./internal/store/...
func TestDynamoStore_RegistrationFlow(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set — skipping LocalStack integration test")
	}

	ctx := context.Background()
	s, err := NewDynamoStoreFromEnv(ctx)
	if err != nil {
		t.Fatalf("build store: %v", err)
	}

	runner := models.Runner{
		ID:        uuid.NewString(),
		Name:      "Integration Test Runner " + uuid.NewString(),
		BirthDate: "1990-01-01",
		Gender:    "X",
		CreatedAt: time.Now().UTC(),
	}

	if err := s.CreateRunner(ctx, runner); err != nil {
		t.Fatalf("create runner: %v", err)
	}

	got, err := s.RunnerByNameDOB(ctx, runner.NameDobKey())
	if err != nil {
		t.Fatalf("lookup runner: %v", err)
	}
	if got == nil || got.ID != runner.ID {
		t.Fatalf("runner lookup mismatch: got=%+v want id=%s", got, runner.ID)
	}

	reg := models.Registration{
		ID:        uuid.NewString(),
		RaceID:    "rc-integration-" + uuid.NewString(),
		RunnerID:  runner.ID,
		Status:    "received",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.CreateRegistration(ctx, reg); err != nil {
		t.Fatalf("create registration: %v", err)
	}

	dup := reg
	dup.ID = uuid.NewString()
	if err := s.CreateRegistration(ctx, dup); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("duplicate registration: want ErrAlreadyRegistered, got %v", err)
	}
}
