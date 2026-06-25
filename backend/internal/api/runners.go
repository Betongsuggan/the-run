package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type RunnerDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Gender    string `json:"gender"`
	BirthYear *int   `json:"birthYear,omitempty"`
}

// birthYearFromDate pulls the leading 4-char year out of a YYYY-MM-DD birth
// date. Returns nil if the field is empty.
func birthYearFromDate(birthDate string) *int {
	if len(birthDate) < 4 {
		return nil
	}
	var y int
	if _, err := fmt.Sscanf(birthDate[:4], "%d", &y); err != nil || y == 0 {
		return nil
	}
	return &y
}

func runnerToDTO(r models.Runner) RunnerDTO {
	return RunnerDTO{
		ID:        r.ID,
		Name:      r.Name,
		Gender:    r.Gender,
		BirthYear: birthYearFromDate(r.BirthDate),
	}
}

type listRunnersOutput struct {
	Body struct {
		Runners []RunnerDTO `json:"runners"`
	}
}

type runnerIDInput struct {
	ID string `path:"id"`
}

type runnerOutput struct {
	Body RunnerDTO
}

type createRunnerInput struct {
	Body struct {
		Name      string `json:"name" minLength:"1" maxLength:"120"`
		Gender    string `json:"gender" enum:"M,F,X"`
		BirthYear *int   `json:"birthYear,omitempty" minimum:"1900" maximum:"2100"`
	}
}

type updateRunnerInput struct {
	ID   string `path:"id"`
	Body struct {
		Name      string `json:"name" minLength:"1" maxLength:"120"`
		Gender    string `json:"gender" enum:"M,F,X"`
		BirthYear *int   `json:"birthYear,omitempty" minimum:"1900" maximum:"2100"`
	}
}

// birthDateFromYear stores `YYYY-01-01` when we only know a birth year (the
// admin form supplies year only). This keeps the byNameDOB GSI key stable
// when admin creates a runner that the public form might later look up.
func birthDateFromYear(year *int) string {
	if year == nil {
		return ""
	}
	return fmt.Sprintf("%04d-01-01", *year)
}

func registerRunners(api huma.API, s store.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-runners",
		Method:      "GET",
		Path:        "/runners",
		Summary:     "List all runners",
		Tags:        []string{"runners"},
	}, func(ctx context.Context, _ *struct{}) (*listRunnersOutput, error) {
		rs, err := s.ListRunners(ctx)
		if err != nil {
			return nil, fmt.Errorf("list runners: %w", err)
		}
		out := &listRunnersOutput{}
		out.Body.Runners = make([]RunnerDTO, len(rs))
		for i, r := range rs {
			out.Body.Runners[i] = runnerToDTO(r)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-runner",
		Method:      "GET",
		Path:        "/runners/{id}",
		Summary:     "Get a runner by ID",
		Tags:        []string{"runners"},
	}, func(ctx context.Context, in *runnerIDInput) (*runnerOutput, error) {
		r, err := s.GetRunner(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("runner not found")
			}
			return nil, fmt.Errorf("get runner: %w", err)
		}
		return &runnerOutput{Body: runnerToDTO(*r)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-runner",
		Method:        "POST",
		Path:          "/runners",
		Summary:       "Create a runner",
		Tags:          []string{"runners"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createRunnerInput) (*runnerOutput, error) {
		birthDate := birthDateFromYear(in.Body.BirthYear)
		name := strings.TrimSpace(in.Body.Name)
		if birthDate != "" {
			existing, err := s.RunnerByNameDOB(ctx, models.NameDobKey(name, birthDate))
			if err != nil {
				return nil, fmt.Errorf("lookup runner: %w", err)
			}
			if existing != nil {
				return nil, huma.Error409Conflict("a runner with that name and birth year already exists")
			}
		}
		r := models.Runner{
			ID:        uuid.NewString(),
			Name:      name,
			BirthDate: birthDate,
			Gender:    in.Body.Gender,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.CreateRunner(ctx, r); err != nil {
			return nil, fmt.Errorf("create runner: %w", err)
		}
		return &runnerOutput{Body: runnerToDTO(r)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-runner",
		Method:      "PUT",
		Path:        "/runners/{id}",
		Summary:     "Update a runner",
		Tags:        []string{"runners"},
	}, func(ctx context.Context, in *updateRunnerInput) (*runnerOutput, error) {
		existing, err := s.GetRunner(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("runner not found")
			}
			return nil, fmt.Errorf("get runner: %w", err)
		}
		updated := models.Runner{
			ID:        existing.ID,
			Name:      strings.TrimSpace(in.Body.Name),
			BirthDate: birthDateFromYear(in.Body.BirthYear),
			Gender:    in.Body.Gender,
			CreatedAt: existing.CreatedAt,
		}
		if err := s.UpdateRunner(ctx, updated); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("runner not found")
			}
			return nil, fmt.Errorf("update runner: %w", err)
		}
		return &runnerOutput{Body: runnerToDTO(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-runner",
		Method:        "DELETE",
		Path:          "/runners/{id}",
		Summary:       "Delete a runner (cascades to registrations)",
		Tags:          []string{"runners"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *runnerIDInput) (*struct{}, error) {
		if err := s.DeleteRunner(ctx, in.ID); err != nil {
			return nil, fmt.Errorf("delete runner: %w", err)
		}
		return nil, nil
	})
}
