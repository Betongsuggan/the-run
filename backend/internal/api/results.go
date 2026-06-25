package api

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type ResultDTO struct {
	ID                string      `json:"id"`
	RaceID            string      `json:"raceId"`
	RunnerID          string      `json:"runnerId"`
	Bib               string      `json:"bib"`
	FinishSeconds     int         `json:"finishSeconds"`
	Category          CategoryDTO `json:"category"`
	PlacementOverall  int         `json:"placementOverall"`
	PlacementCategory int         `json:"placementCategory"`
	Splits            []SplitDTO  `json:"splits,omitempty"`
	Conditions        string      `json:"conditions,omitempty"`
	Notes             string      `json:"notes,omitempty"`
}

type ResultExpandedDTO struct {
	ResultDTO
	Race   RaceDTO   `json:"race"`
	Event  EventDTO  `json:"event"`
	Runner RunnerDTO `json:"runner"`
}

type listResultsByRaceInput struct {
	RaceID string `path:"raceId"`
}

type listResultsOutput struct {
	Body struct {
		Results []ResultExpandedDTO `json:"results"`
	}
}

type listResultsByRunnerInput struct {
	RunnerID string `path:"runnerId"`
}

type resultIDInput struct {
	ID string `path:"id"`
}

type resultOutput struct {
	Body ResultExpandedDTO
}

func categoryKey(c *models.Category) string {
	if c == nil {
		return "|"
	}
	return c.Gender + "|" + c.AgeGroup
}

// computeResultsForRace sorts the race's finished registrations by
// finishSeconds ascending and assigns placementOverall + placementCategory.
func computeResultsForRace(regs []models.Registration) []ResultDTO {
	finished := make([]models.Registration, 0, len(regs))
	for _, r := range regs {
		if r.Status == models.StatusFinished && r.FinishSeconds != nil {
			finished = append(finished, r)
		}
	}
	sort.Slice(finished, func(i, j int) bool { return *finished[i].FinishSeconds < *finished[j].FinishSeconds })

	catCount := map[string]int{}
	out := make([]ResultDTO, len(finished))
	for i, r := range finished {
		key := categoryKey(r.Category)
		catCount[key]++
		var category CategoryDTO
		if r.Category != nil {
			category = CategoryDTO{Gender: r.Category.Gender, AgeGroup: r.Category.AgeGroup}
		}
		splits := make([]SplitDTO, len(r.Splits))
		for j, sp := range r.Splits {
			splits[j] = SplitDTO{Km: sp.Km, TimeSeconds: sp.TimeSeconds}
		}
		out[i] = ResultDTO{
			ID:                r.ID,
			RaceID:            r.RaceID,
			RunnerID:          r.RunnerID,
			Bib:               r.Bib,
			FinishSeconds:     *r.FinishSeconds,
			Category:          category,
			PlacementOverall:  i + 1,
			PlacementCategory: catCount[key],
			Splits:            splits,
			Conditions:        r.Conditions,
			Notes:             r.Notes,
		}
	}
	return out
}

type raceContext struct {
	race   models.Race
	event  models.Event
	runner func(id string) (*models.Runner, error)
}

func expandResult(res ResultDTO, ctx raceContext) (*ResultExpandedDTO, error) {
	runner, err := ctx.runner(res.RunnerID)
	if err != nil {
		return nil, err
	}
	return &ResultExpandedDTO{
		ResultDTO: res,
		Race:      raceToDTO(ctx.race),
		Event:     eventToDTO(ctx.event),
		Runner:    runnerToDTO(*runner),
	}, nil
}

// loadRaceContext fetches the race + event so we can expand results without
// re-querying for each registration in the race.
func loadRaceContext(ctx context.Context, s store.Store, raceID string) (*raceContext, error) {
	race, err := s.GetRace(ctx, raceID)
	if err != nil {
		return nil, err
	}
	event, err := s.GetEvent(ctx, race.EventID)
	if err != nil {
		return nil, err
	}
	cache := map[string]*models.Runner{}
	return &raceContext{
		race:  *race,
		event: *event,
		runner: func(id string) (*models.Runner, error) {
			if r, ok := cache[id]; ok {
				return r, nil
			}
			r, err := s.GetRunner(ctx, id)
			if err != nil {
				return nil, err
			}
			cache[id] = r
			return r, nil
		},
	}, nil
}

func registerResults(api huma.API, s store.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-results-by-race",
		Method:      "GET",
		Path:        "/races/{raceId}/results",
		Summary:     "List computed results for a race",
		Tags:        []string{"results"},
	}, func(ctx context.Context, in *listResultsByRaceInput) (*listResultsOutput, error) {
		regs, err := s.ListRegistrationsByRace(ctx, in.RaceID)
		if err != nil {
			return nil, fmt.Errorf("list registrations: %w", err)
		}
		results := computeResultsForRace(regs)
		if len(results) == 0 {
			out := &listResultsOutput{}
			out.Body.Results = []ResultExpandedDTO{}
			return out, nil
		}
		rc, err := loadRaceContext(ctx, s, in.RaceID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("race or its event not found")
			}
			return nil, fmt.Errorf("load race context: %w", err)
		}
		out := &listResultsOutput{}
		out.Body.Results = make([]ResultExpandedDTO, 0, len(results))
		for _, r := range results {
			expanded, err := expandResult(r, *rc)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("expand result: %w", err)
			}
			out.Body.Results = append(out.Body.Results, *expanded)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-results-by-runner",
		Method:      "GET",
		Path:        "/runners/{runnerId}/results",
		Summary:     "List a runner's results across all races",
		Tags:        []string{"results"},
	}, func(ctx context.Context, in *listResultsByRunnerInput) (*listResultsOutput, error) {
		regs, err := s.ListRegistrationsByRunner(ctx, in.RunnerID)
		if err != nil {
			return nil, fmt.Errorf("list registrations: %w", err)
		}
		raceIDs := uniqueRaceIDs(regs)
		expanded := make([]ResultExpandedDTO, 0, len(raceIDs))
		for _, raceID := range raceIDs {
			raceRegs, err := s.ListRegistrationsByRace(ctx, raceID)
			if err != nil {
				return nil, fmt.Errorf("list registrations by race: %w", err)
			}
			results := computeResultsForRace(raceRegs)
			rc, err := loadRaceContext(ctx, s, raceID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("load race context: %w", err)
			}
			for _, r := range results {
				if r.RunnerID != in.RunnerID {
					continue
				}
				e, err := expandResult(r, *rc)
				if err != nil {
					if errors.Is(err, store.ErrNotFound) {
						continue
					}
					return nil, fmt.Errorf("expand result: %w", err)
				}
				expanded = append(expanded, *e)
			}
		}
		sort.Slice(expanded, func(i, j int) bool { return expanded[i].Event.Date > expanded[j].Event.Date })
		out := &listResultsOutput{}
		out.Body.Results = expanded
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-result",
		Method:      "GET",
		Path:        "/results/{id}",
		Summary:     "Get a single expanded result by registration ID",
		Tags:        []string{"results"},
	}, func(ctx context.Context, in *resultIDInput) (*resultOutput, error) {
		reg, err := s.GetRegistrationByID(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("result not found")
			}
			return nil, fmt.Errorf("get registration: %w", err)
		}
		if reg.Status != models.StatusFinished || reg.FinishSeconds == nil {
			return nil, huma.Error404NotFound("result not found")
		}
		raceRegs, err := s.ListRegistrationsByRace(ctx, reg.RaceID)
		if err != nil {
			return nil, fmt.Errorf("list registrations by race: %w", err)
		}
		rc, err := loadRaceContext(ctx, s, reg.RaceID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("race or event not found")
			}
			return nil, fmt.Errorf("load race context: %w", err)
		}
		for _, r := range computeResultsForRace(raceRegs) {
			if r.ID == in.ID {
				e, err := expandResult(r, *rc)
				if err != nil {
					return nil, fmt.Errorf("expand result: %w", err)
				}
				return &resultOutput{Body: *e}, nil
			}
		}
		return nil, huma.Error404NotFound("result not found")
	})
}

func uniqueRaceIDs(regs []models.Registration) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, r := range regs {
		if _, ok := seen[r.RaceID]; ok {
			continue
		}
		seen[r.RaceID] = struct{}{}
		out = append(out, r.RaceID)
	}
	return out
}
