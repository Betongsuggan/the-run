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

	"github.com/BirgerRydback/the-run/backend/internal/audit"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type CategoryDTO struct {
	Gender   string `json:"gender,omitempty"`
	AgeGroup string `json:"ageGroup,omitempty"`
}

type SplitDTO struct {
	Km          int `json:"km"`
	TimeSeconds int `json:"timeSeconds"`
}

type RegistrationDTO struct {
	ID                string       `json:"id"`
	RaceID            string       `json:"raceId"`
	RunnerID          string       `json:"runnerId"`
	Bib               string       `json:"bib,omitempty"`
	Category          *CategoryDTO `json:"category,omitempty"`
	Status            string       `json:"status"`
	FinishSeconds     *int         `json:"finishSeconds,omitempty"`
	Splits            []SplitDTO   `json:"splits,omitempty"`
	Conditions        string       `json:"conditions,omitempty"`
	Notes             string       `json:"notes,omitempty"`
	PaymentReceivedAt string       `json:"paymentReceivedAt,omitempty"`
}

func registrationToDTO(r models.Registration) RegistrationDTO {
	dto := RegistrationDTO{
		ID:         r.ID,
		RaceID:     r.RaceID,
		RunnerID:   r.RunnerID,
		Bib:        r.Bib,
		Status:     r.Status,
		Conditions: r.Conditions,
		Notes:      r.Notes,
	}
	if r.Category != nil {
		dto.Category = &CategoryDTO{Gender: r.Category.Gender, AgeGroup: r.Category.AgeGroup}
	}
	if r.FinishSeconds != nil {
		v := *r.FinishSeconds
		dto.FinishSeconds = &v
	}
	if len(r.Splits) > 0 {
		dto.Splits = make([]SplitDTO, len(r.Splits))
		for i, s := range r.Splits {
			dto.Splits[i] = SplitDTO{Km: s.Km, TimeSeconds: s.TimeSeconds}
		}
	}
	if r.PaymentReceivedAt != nil {
		dto.PaymentReceivedAt = r.PaymentReceivedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

type listRegistrationsByRaceInput struct {
	RaceID string `path:"raceId"`
}

type listRegistrationsOutput struct {
	Body struct {
		Registrations []RegistrationDTO `json:"registrations"`
	}
}

type registrationOutput struct {
	Body RegistrationDTO
}

type registrationKeyInput struct {
	RaceID   string `path:"raceId"`
	RunnerID string `path:"runnerId"`
}

type createAdminRegistrationInput struct {
	Body struct {
		RaceID   string       `json:"raceId" minLength:"1"`
		RunnerID string       `json:"runnerId" minLength:"1"`
		Bib      string       `json:"bib,omitempty" maxLength:"40"`
		Category *CategoryDTO `json:"category,omitempty"`
	}
}

type registrationFields struct {
	Status            *string      `json:"status,omitempty" enum:"pending,finished,dnf,dns"`
	Bib               *string      `json:"bib,omitempty"`
	Category          *CategoryDTO `json:"category,omitempty"`
	FinishSeconds     *int         `json:"finishSeconds,omitempty"`
	ClearFinish       bool         `json:"clearFinish,omitempty"`
	Splits            []SplitDTO   `json:"splits,omitempty"`
	ClearSplits       bool         `json:"clearSplits,omitempty"`
	Conditions        *string      `json:"conditions,omitempty"`
	Notes             *string      `json:"notes,omitempty"`
	PaymentReceivedAt *string      `json:"paymentReceivedAt,omitempty" doc:"RFC3339 timestamp to mark payment received; empty string to clear; omitted to leave unchanged"`
}

type updateRegistrationInput struct {
	RaceID   string `path:"raceId"`
	RunnerID string `path:"runnerId"`
	Body     registrationFields
}

type bulkRegistrationUpdate struct {
	RaceID            string       `json:"raceId" minLength:"1"`
	RunnerID          string       `json:"runnerId" minLength:"1"`
	Status            *string      `json:"status,omitempty" enum:"pending,finished,dnf,dns"`
	Bib               *string      `json:"bib,omitempty"`
	Category          *CategoryDTO `json:"category,omitempty"`
	FinishSeconds     *int         `json:"finishSeconds,omitempty"`
	ClearFinish       bool         `json:"clearFinish,omitempty"`
	Splits            []SplitDTO   `json:"splits,omitempty"`
	ClearSplits       bool         `json:"clearSplits,omitempty"`
	Conditions        *string      `json:"conditions,omitempty"`
	Notes             *string      `json:"notes,omitempty"`
	PaymentReceivedAt *string      `json:"paymentReceivedAt,omitempty"`
}

func (b bulkRegistrationUpdate) fields() registrationFields {
	return registrationFields{
		Status:            b.Status,
		Bib:               b.Bib,
		Category:          b.Category,
		FinishSeconds:     b.FinishSeconds,
		ClearFinish:       b.ClearFinish,
		Splits:            b.Splits,
		ClearSplits:       b.ClearSplits,
		Conditions:        b.Conditions,
		Notes:             b.Notes,
		PaymentReceivedAt: b.PaymentReceivedAt,
	}
}

type bulkUpdateRegistrationInput struct {
	Body struct {
		Updates []bulkRegistrationUpdate `json:"updates"`
	}
}

type bulkUpdateRegistrationOutput struct {
	Body struct {
		Registrations []RegistrationDTO `json:"registrations"`
	}
}

func toStoreUpdate(raceID, runnerID string, f registrationFields) (store.RegistrationUpdate, error) {
	u := store.RegistrationUpdate{
		RaceID:        raceID,
		RunnerID:      runnerID,
		Status:        f.Status,
		Bib:           f.Bib,
		FinishSeconds: f.FinishSeconds,
		ClearFinish:   f.ClearFinish,
		ClearSplits:   f.ClearSplits,
		Conditions:    f.Conditions,
		Notes:         f.Notes,
	}
	if f.Category != nil {
		u.Category = &models.Category{Gender: f.Category.Gender, AgeGroup: f.Category.AgeGroup}
	}
	if f.Splits != nil {
		u.Splits = make([]models.Split, len(f.Splits))
		for i, s := range f.Splits {
			u.Splits[i] = models.Split{Km: s.Km, TimeSeconds: s.TimeSeconds}
		}
	}
	if f.PaymentReceivedAt != nil {
		// Empty string = clear the field; non-empty = parse as RFC3339 (or
		// accept the sentinel "now" so the client doesn't need a clock).
		raw := strings.TrimSpace(*f.PaymentReceivedAt)
		switch raw {
		case "":
			u.ClearPayment = true
		case "now":
			now := time.Now().UTC()
			u.PaymentReceivedAt = &now
		default:
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return u, fmt.Errorf("paymentReceivedAt must be RFC3339 or empty: %w", err)
			}
			u.PaymentReceivedAt = &t
		}
	}
	if f.Status != nil && *f.Status == models.StatusFinished {
		// finished requires a finishSeconds value, either already on the
		// record or supplied in this update.
		if f.FinishSeconds == nil {
			return u, errors.New("status=finished requires finishSeconds")
		}
	}
	return u, nil
}

func registerAdminRegistrations(api huma.API, s store.Store, authCfg auth.Config, recorder audit.Recorder) {
	adminMW := adminMiddlewares(authCfg)
	// auditAdminAction records an audit row attributed to the calling
	// admin, against the runner whose registration was touched. Looks up
	// the runner to find AccountID — without that we'd have no partition
	// key. Silent on lookup failure (best-effort observability).
	auditAdminAction := func(ctx context.Context, raceID, runnerID, action, summary string) {
		runner, err := s.GetRunner(ctx, runnerID)
		if err != nil || runner == nil {
			return
		}
		actor := "admin"
		if sub := auth.Subject(ctx); sub != "" {
			actor = "admin:" + sub
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  runner.AccountID,
			Action:     action,
			Actor:      actor,
			TargetType: "registration",
			TargetID:   raceID + "/" + runnerID,
			Summary:    summary,
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "list-all-registrations",
		Method:      "GET",
		Path:        "/registrations",
		Summary:     "List all registrations (admin)",
		Tags:        []string{"registrations"},
		Middlewares: adminMW,
	}, func(ctx context.Context, _ *struct{}) (*listRegistrationsOutput, error) {
		regs, err := s.ListRegistrations(ctx)
		if err != nil {
			return nil, fmt.Errorf("list registrations: %w", err)
		}
		out := &listRegistrationsOutput{}
		out.Body.Registrations = make([]RegistrationDTO, len(regs))
		for i, r := range regs {
			out.Body.Registrations[i] = registrationToDTO(r)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-registrations-by-race",
		Method:      "GET",
		Path:        "/races/{raceId}/registrations",
		Summary:     "List registrations for a race",
		Tags:        []string{"registrations"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *listRegistrationsByRaceInput) (*listRegistrationsOutput, error) {
		regs, err := s.ListRegistrationsByRace(ctx, in.RaceID)
		if err != nil {
			return nil, fmt.Errorf("list registrations: %w", err)
		}
		out := &listRegistrationsOutput{}
		out.Body.Registrations = make([]RegistrationDTO, len(regs))
		for i, r := range regs {
			out.Body.Registrations[i] = registrationToDTO(r)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-create-registration",
		Method:        "POST",
		Path:          "/registrations/admin",
		Summary:       "Create a registration (admin)",
		Description:   "Admin-side counterpart to POST /registrations. Skips the public form's honeypot and date-of-birth validation, since the runner already exists.",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *createAdminRegistrationInput) (*registrationOutput, error) {
		if _, err := s.GetRace(ctx, in.Body.RaceID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error422UnprocessableEntity("raceId does not reference an existing race")
			}
			return nil, fmt.Errorf("lookup race: %w", err)
		}
		runner, err := s.GetRunner(ctx, in.Body.RunnerID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error422UnprocessableEntity("runnerId does not reference an existing runner")
			}
			return nil, fmt.Errorf("lookup runner: %w", err)
		}
		category := &models.Category{Gender: runner.Gender}
		if in.Body.Category != nil {
			category = &models.Category{Gender: in.Body.Category.Gender, AgeGroup: in.Body.Category.AgeGroup}
		}
		reg := models.Registration{
			ID:        uuid.NewString(),
			RaceID:    in.Body.RaceID,
			RunnerID:  runner.ID,
			Status:    models.StatusPending,
			Bib:       strings.TrimSpace(in.Body.Bib),
			Category:  category,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.CreateRegistration(ctx, reg); err != nil {
			if errors.Is(err, store.ErrAlreadyRegistered) {
				return nil, huma.Error409Conflict("this runner is already registered for this race")
			}
			return nil, fmt.Errorf("create registration: %w", err)
		}
		auditAdminAction(ctx, reg.RaceID, reg.RunnerID, "admin.registration.created", "Admin created registration")
		return &registrationOutput{Body: registrationToDTO(reg)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-registration",
		Method:      "PATCH",
		Path:        "/races/{raceId}/registrations/{runnerId}",
		Summary:     "Update fields on a single registration",
		Tags:        []string{"registrations"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *updateRegistrationInput) (*registrationOutput, error) {
		u, err := toStoreUpdate(in.RaceID, in.RunnerID, in.Body)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		reg, err := s.UpdateRegistration(ctx, u)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("registration not found")
			}
			return nil, fmt.Errorf("update registration: %w", err)
		}
		auditAdminAction(ctx, reg.RaceID, reg.RunnerID, "admin.registration.updated", "Admin updated registration")
		return &registrationOutput{Body: registrationToDTO(*reg)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bulk-update-registrations",
		Method:      "PATCH",
		Path:        "/registrations/bulk",
		Summary:     "Update multiple registrations in one call",
		Tags:        []string{"registrations"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *bulkUpdateRegistrationInput) (*bulkUpdateRegistrationOutput, error) {
		results := make([]RegistrationDTO, 0, len(in.Body.Updates))
		for _, raw := range in.Body.Updates {
			u, err := toStoreUpdate(raw.RaceID, raw.RunnerID, raw.fields())
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("%s/%s: %s", raw.RaceID, raw.RunnerID, err.Error()))
			}
			reg, err := s.UpdateRegistration(ctx, u)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound(fmt.Sprintf("registration %s/%s not found", raw.RaceID, raw.RunnerID))
				}
				return nil, fmt.Errorf("update registration %s/%s: %w", raw.RaceID, raw.RunnerID, err)
			}
			results = append(results, registrationToDTO(*reg))
		}
		out := &bulkUpdateRegistrationOutput{}
		out.Body.Registrations = results
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-registration",
		Method:        "DELETE",
		Path:          "/races/{raceId}/registrations/{runnerId}",
		Summary:       "Delete a registration",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusNoContent,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *registrationKeyInput) (*struct{}, error) {
		// Audit BEFORE the delete so we can still resolve the runner →
		// accountId. After the delete the runner might still exist, but
		// future deletes (e.g. cascading runner erasure) could leave us
		// without a key.
		auditAdminAction(ctx, in.RaceID, in.RunnerID, "admin.registration.deleted", "Admin deleted registration")
		if err := s.DeleteRegistration(ctx, in.RaceID, in.RunnerID); err != nil {
			return nil, fmt.Errorf("delete registration: %w", err)
		}
		return nil, nil
	})
}
