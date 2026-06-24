package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type RegisterInput struct {
	Body struct {
		Name        string `json:"name" minLength:"1" maxLength:"120" doc:"Full name of the registrant"`
		DateOfBirth string `json:"dateOfBirth" format:"date" doc:"Birth date (YYYY-MM-DD)"`
		Gender      string `json:"gender" enum:"M,F,X" doc:"Gender code"`
		RaceID      string `json:"raceId" minLength:"1" doc:"ID of the race to register for"`
		// Honeypot field: legitimate clients leave it empty. Real bot
		// protection (Turnstile / hCaptcha) is a TODO — see PROJECT_PLAN.md.
		Website string `json:"website,omitempty" doc:"Leave blank — honeypot"`
	}
}

type RegisterOutput struct {
	Body struct {
		ID       string `json:"id" doc:"Server-assigned registration ID"`
		RunnerID string `json:"runnerId" doc:"ID of the runner (existing or newly created)"`
		Status   string `json:"status" doc:"Lifecycle status, e.g. 'received'"`
	}
}

func registerRegistrations(api huma.API, s store.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "register-for-race",
		Method:        "POST",
		Path:          "/registrations",
		Summary:       "Register for a race",
		Description:   "Public race registration. No authentication required; the race must belong to an event in the future.",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *RegisterInput) (*RegisterOutput, error) {
		if strings.TrimSpace(in.Body.Website) != "" {
			// Honeypot tripped — pretend success so the bot doesn't probe further.
			out := &RegisterOutput{}
			out.Body.ID = "ignored"
			out.Body.Status = "received"
			return out, nil
		}

		dob, err := time.Parse("2006-01-02", in.Body.DateOfBirth)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth must be YYYY-MM-DD")
		}
		if dob.After(time.Now()) {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth cannot be in the future")
		}

		// TODO(B1): once race + event tables exist, validate that the race
		// exists and its event date is today or later. For now we trust the
		// raceId — the frontend filters to upcoming races.

		nameDobKey := models.NameDobKey(in.Body.Name, in.Body.DateOfBirth)
		runner, err := s.RunnerByNameDOB(ctx, nameDobKey)
		if err != nil {
			return nil, fmt.Errorf("lookup runner: %w", err)
		}
		if runner == nil {
			runner = &models.Runner{
				ID:        uuid.NewString(),
				Name:      strings.TrimSpace(in.Body.Name),
				BirthDate: in.Body.DateOfBirth,
				Gender:    in.Body.Gender,
				CreatedAt: time.Now().UTC(),
			}
			if err := s.CreateRunner(ctx, *runner); err != nil {
				return nil, fmt.Errorf("create runner: %w", err)
			}
		}

		reg := models.Registration{
			ID:        uuid.NewString(),
			RaceID:    in.Body.RaceID,
			RunnerID:  runner.ID,
			Status:    "received",
			CreatedAt: time.Now().UTC(),
		}
		if err := s.CreateRegistration(ctx, reg); err != nil {
			if errors.Is(err, store.ErrAlreadyRegistered) {
				return nil, huma.Error409Conflict("already registered for this race")
			}
			return nil, fmt.Errorf("create registration: %w", err)
		}

		log.Printf("registration stored: id=%s runnerId=%s raceId=%s", reg.ID, runner.ID, reg.RaceID)

		out := &RegisterOutput{}
		out.Body.ID = reg.ID
		out.Body.RunnerID = runner.ID
		out.Body.Status = reg.Status
		return out, nil
	})
}
