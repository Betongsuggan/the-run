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

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type RaceDTO struct {
	ID                 string `json:"id"`
	EventID            string `json:"eventId"`
	Name               string `json:"name"`
	DistanceMeters     int    `json:"distanceMeters"`
	Discipline         string `json:"discipline"`
	MaxRunners         int    `json:"maxRunners"`
	RegistrationFeeOre int    `json:"registrationFeeOre"`
}

func raceToDTO(r models.Race) RaceDTO {
	return RaceDTO{
		ID:                 r.ID,
		EventID:            r.EventID,
		Name:               r.Name,
		DistanceMeters:     r.DistanceMeters,
		Discipline:         r.Discipline,
		MaxRunners:         r.MaxRunners,
		RegistrationFeeOre: r.RegistrationFeeOre,
	}
}

type listRacesOutput struct {
	Body struct {
		Races []RaceDTO `json:"races"`
	}
}

type listRacesByEventInput struct {
	EventID string `path:"eventId"`
}

type raceIDInput struct {
	ID string `path:"id"`
}

type raceOutput struct {
	Body RaceDTO
}

type createRaceInput struct {
	Body struct {
		EventID            string `json:"eventId" minLength:"1"`
		Name               string `json:"name" minLength:"1" maxLength:"200"`
		DistanceMeters     int    `json:"distanceMeters" minimum:"1"`
		Discipline         string `json:"discipline" enum:"run,walk,kids"`
		MaxRunners         int    `json:"maxRunners,omitempty" minimum:"0"`
		RegistrationFeeOre int    `json:"registrationFeeOre,omitempty" minimum:"0"`
	}
}

type updateRaceInput struct {
	ID   string `path:"id"`
	Body struct {
		EventID            string `json:"eventId" minLength:"1"`
		Name               string `json:"name" minLength:"1" maxLength:"200"`
		DistanceMeters     int    `json:"distanceMeters" minimum:"1"`
		Discipline         string `json:"discipline" enum:"run,walk,kids"`
		MaxRunners         int    `json:"maxRunners,omitempty" minimum:"0"`
		RegistrationFeeOre int    `json:"registrationFeeOre,omitempty" minimum:"0"`
	}
}

func registerRaces(api huma.API, s store.Store, authCfg auth.Config) {
	adminMW := adminMiddlewares(authCfg)

	huma.Register(api, huma.Operation{
		OperationID: "list-races",
		Method:      "GET",
		Path:        "/races",
		Summary:     "List all races",
		Tags:        []string{"races"},
	}, func(ctx context.Context, _ *struct{}) (*listRacesOutput, error) {
		races, err := s.ListRaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("list races: %w", err)
		}
		out := &listRacesOutput{}
		out.Body.Races = make([]RaceDTO, len(races))
		for i, r := range races {
			out.Body.Races[i] = raceToDTO(r)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-races-by-event",
		Method:      "GET",
		Path:        "/events/{eventId}/races",
		Summary:     "List races for an event",
		Tags:        []string{"races"},
	}, func(ctx context.Context, in *listRacesByEventInput) (*listRacesOutput, error) {
		races, err := s.ListRacesByEvent(ctx, in.EventID)
		if err != nil {
			return nil, fmt.Errorf("list races by event: %w", err)
		}
		out := &listRacesOutput{}
		out.Body.Races = make([]RaceDTO, len(races))
		for i, r := range races {
			out.Body.Races[i] = raceToDTO(r)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-race",
		Method:      "GET",
		Path:        "/races/{id}",
		Summary:     "Get a race by ID",
		Tags:        []string{"races"},
	}, func(ctx context.Context, in *raceIDInput) (*raceOutput, error) {
		r, err := s.GetRace(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("race not found")
			}
			return nil, fmt.Errorf("get race: %w", err)
		}
		return &raceOutput{Body: raceToDTO(*r)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-race",
		Method:        "POST",
		Path:          "/races",
		Summary:       "Create a race",
		Tags:          []string{"races"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *createRaceInput) (*raceOutput, error) {
		if _, err := s.GetEvent(ctx, in.Body.EventID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error422UnprocessableEntity("eventId does not reference an existing event")
			}
			return nil, fmt.Errorf("lookup event: %w", err)
		}
		r := models.Race{
			ID:                 uuid.NewString(),
			EventID:            in.Body.EventID,
			Name:               strings.TrimSpace(in.Body.Name),
			DistanceMeters:     in.Body.DistanceMeters,
			Discipline:         in.Body.Discipline,
			MaxRunners:         in.Body.MaxRunners,
			RegistrationFeeOre: in.Body.RegistrationFeeOre,
			CreatedAt:          time.Now().UTC(),
		}
		if err := s.CreateRace(ctx, r); err != nil {
			return nil, fmt.Errorf("create race: %w", err)
		}
		return &raceOutput{Body: raceToDTO(r)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-race",
		Method:      "PUT",
		Path:        "/races/{id}",
		Summary:     "Update a race",
		Tags:        []string{"races"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *updateRaceInput) (*raceOutput, error) {
		existing, err := s.GetRace(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("race not found")
			}
			return nil, fmt.Errorf("get race: %w", err)
		}
		if in.Body.EventID != existing.EventID {
			if _, err := s.GetEvent(ctx, in.Body.EventID); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error422UnprocessableEntity("eventId does not reference an existing event")
				}
				return nil, fmt.Errorf("lookup event: %w", err)
			}
		}
		updated := models.Race{
			ID:                 existing.ID,
			EventID:            in.Body.EventID,
			Name:               strings.TrimSpace(in.Body.Name),
			DistanceMeters:     in.Body.DistanceMeters,
			Discipline:         in.Body.Discipline,
			MaxRunners:         in.Body.MaxRunners,
			RegistrationFeeOre: in.Body.RegistrationFeeOre,
			CreatedAt:          existing.CreatedAt,
		}
		if err := s.UpdateRace(ctx, updated); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("race not found")
			}
			return nil, fmt.Errorf("update race: %w", err)
		}
		return &raceOutput{Body: raceToDTO(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-race",
		Method:        "DELETE",
		Path:          "/races/{id}",
		Summary:       "Delete a race (cascades to registrations)",
		Tags:          []string{"races"},
		DefaultStatus: http.StatusNoContent,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *raceIDInput) (*struct{}, error) {
		if err := s.DeleteRace(ctx, in.ID); err != nil {
			return nil, fmt.Errorf("delete race: %w", err)
		}
		return nil, nil
	})
}
