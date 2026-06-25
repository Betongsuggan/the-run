package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type EventDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Year     int    `json:"year"`
	Date     string `json:"date"`
	Location string `json:"location,omitempty"`
}

func eventToDTO(e models.Event) EventDTO {
	return EventDTO{
		ID:       e.ID,
		Name:     e.Name,
		Year:     e.Year,
		Date:     e.Date,
		Location: e.Location,
	}
}

type listEventsOutput struct {
	Body struct {
		Events []EventDTO `json:"events"`
	}
}

type getEventInput struct {
	ID string `path:"id"`
}

type eventOutput struct {
	Body EventDTO
}

type createEventInput struct {
	Body struct {
		Name     string `json:"name" minLength:"1" maxLength:"200"`
		Year     int    `json:"year" minimum:"2000" maximum:"3000"`
		Date     string `json:"date" format:"date"`
		Location string `json:"location,omitempty" maxLength:"200"`
	}
}

type updateEventInput struct {
	ID   string `path:"id"`
	Body struct {
		Name     string `json:"name" minLength:"1" maxLength:"200"`
		Year     int    `json:"year" minimum:"2000" maximum:"3000"`
		Date     string `json:"date" format:"date"`
		Location string `json:"location,omitempty" maxLength:"200"`
	}
}

func registerEvents(api huma.API, s store.Store, authCfg auth.Config) {
	adminMW := adminMiddlewares(authCfg)

	huma.Register(api, huma.Operation{
		OperationID: "list-events",
		Method:      "GET",
		Path:        "/events",
		Summary:     "List all events",
		Tags:        []string{"events"},
	}, func(ctx context.Context, _ *struct{}) (*listEventsOutput, error) {
		events, err := s.ListEvents(ctx)
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		sort.Slice(events, func(i, j int) bool { return events[i].Date > events[j].Date })
		out := &listEventsOutput{}
		out.Body.Events = make([]EventDTO, len(events))
		for i, e := range events {
			out.Body.Events[i] = eventToDTO(e)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-event",
		Method:      "GET",
		Path:        "/events/{id}",
		Summary:     "Get an event by ID",
		Tags:        []string{"events"},
	}, func(ctx context.Context, in *getEventInput) (*eventOutput, error) {
		e, err := s.GetEvent(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("event not found")
			}
			return nil, fmt.Errorf("get event: %w", err)
		}
		return &eventOutput{Body: eventToDTO(*e)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-event",
		Method:        "POST",
		Path:          "/events",
		Summary:       "Create an event",
		Tags:          []string{"events"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *createEventInput) (*eventOutput, error) {
		e := models.Event{
			ID:        uuid.NewString(),
			Name:      strings.TrimSpace(in.Body.Name),
			Year:      in.Body.Year,
			Date:      in.Body.Date,
			Location:  strings.TrimSpace(in.Body.Location),
			CreatedAt: time.Now().UTC(),
		}
		if err := s.CreateEvent(ctx, e); err != nil {
			return nil, fmt.Errorf("create event: %w", err)
		}
		return &eventOutput{Body: eventToDTO(e)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-event",
		Method:      "PUT",
		Path:        "/events/{id}",
		Summary:     "Update an event",
		Tags:        []string{"events"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *updateEventInput) (*eventOutput, error) {
		existing, err := s.GetEvent(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("event not found")
			}
			return nil, fmt.Errorf("get event: %w", err)
		}
		updated := models.Event{
			ID:        existing.ID,
			Name:      strings.TrimSpace(in.Body.Name),
			Year:      in.Body.Year,
			Date:      in.Body.Date,
			Location:  strings.TrimSpace(in.Body.Location),
			CreatedAt: existing.CreatedAt,
		}
		if err := s.UpdateEvent(ctx, updated); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("event not found")
			}
			return nil, fmt.Errorf("update event: %w", err)
		}
		return &eventOutput{Body: eventToDTO(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-event",
		Method:        "DELETE",
		Path:          "/events/{id}",
		Summary:       "Delete an event (cascades to races and registrations)",
		Tags:          []string{"events"},
		DefaultStatus: http.StatusNoContent,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *getEventInput) (*struct{}, error) {
		if err := s.DeleteEvent(ctx, in.ID); err != nil {
			return nil, fmt.Errorf("delete event: %w", err)
		}
		return nil, nil
	})
}
