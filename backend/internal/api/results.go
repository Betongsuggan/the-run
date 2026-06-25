package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type ResultDTO struct {
	ID       string `json:"id"`
	RaceID   string `json:"raceId"`
	RunnerID string `json:"runnerId"`
	// Status mirrors the underlying Registration status: "finished" rows
	// carry FinishSeconds + placement; "dnf"/"dns" rows have neither (the
	// finish-time field is nil). Frontend branches on this to render a
	// badge for non-finished rows.
	Status            string      `json:"status"`
	Bib               string      `json:"bib"`
	FinishSeconds     *int        `json:"finishSeconds,omitempty"`
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

// computeResultsForRace sorts the race's terminal registrations into a
// leaderboard:
//
//  1. StatusFinished rows by finishSeconds ascending, with placementOverall
//     + placementCategory assigned.
//  2. StatusDNF rows next (no placement, no finish time).
//  3. StatusDNS rows last (no placement, no finish time).
//
// Non-terminal statuses (received, pending, pending_guardian_consent,
// pending_deletion) are excluded — they're either "race hasn't happened
// yet" or "in-flight admin state".
//
// Callers can filter further downstream (e.g. expandResult drops DNS rows
// for non-admin viewers).
func computeResultsForRace(regs []models.Registration) []ResultDTO {
	finished := make([]models.Registration, 0, len(regs))
	dnf := make([]models.Registration, 0)
	dns := make([]models.Registration, 0)
	for _, r := range regs {
		switch r.Status {
		case models.StatusFinished:
			if r.FinishSeconds != nil {
				finished = append(finished, r)
			}
		case models.StatusDNF:
			dnf = append(dnf, r)
		case models.StatusDNS:
			dns = append(dns, r)
		}
	}
	sort.Slice(finished, func(i, j int) bool { return *finished[i].FinishSeconds < *finished[j].FinishSeconds })

	catCount := map[string]int{}
	out := make([]ResultDTO, 0, len(finished)+len(dnf)+len(dns))
	for i, r := range finished {
		key := categoryKey(r.Category)
		catCount[key]++
		fs := *r.FinishSeconds
		out = append(out, ResultDTO{
			ID:                r.ID,
			RaceID:            r.RaceID,
			RunnerID:          r.RunnerID,
			Status:            r.Status,
			Bib:               r.Bib,
			FinishSeconds:     &fs,
			Category:          categoryDTO(r.Category),
			PlacementOverall:  i + 1,
			PlacementCategory: catCount[key],
			Splits:            splitsToDTO(r.Splits),
			Conditions:        r.Conditions,
			Notes:             r.Notes,
		})
	}
	for _, group := range [][]models.Registration{dnf, dns} {
		for _, r := range group {
			out = append(out, ResultDTO{
				ID:       r.ID,
				RaceID:   r.RaceID,
				RunnerID: r.RunnerID,
				Status:   r.Status,
				Bib:      r.Bib,
				Category: categoryDTO(r.Category),
				// Placement + FinishSeconds intentionally zero/nil — these
				// runners have no leaderboard position by definition.
			})
		}
	}
	return out
}

func categoryDTO(c *models.Category) CategoryDTO {
	if c == nil {
		return CategoryDTO{}
	}
	return CategoryDTO{Gender: c.Gender, AgeGroup: c.AgeGroup}
}

func splitsToDTO(splits []models.Split) []SplitDTO {
	out := make([]SplitDTO, len(splits))
	for j, sp := range splits {
		out[j] = SplitDTO{Km: sp.Km, TimeSeconds: sp.TimeSeconds}
	}
	return out
}

type raceContext struct {
	race   models.Race
	event  models.Event
	runner func(id string) (*models.Runner, error)
}

// errResultRedactedAway is a sentinel for results that should be silently
// dropped from a listing. Reserved for cases where even an anonymized row
// is the wrong outcome (e.g. some future "race never happened" purge). Kept
// because uniqueRaceIDs callers already handle it.
var errResultRedactedAway = errors.New("result row hidden")

func expandResult(reqCtx context.Context, res ResultDTO, rctx raceContext) (*ResultExpandedDTO, error) {
	runner, err := rctx.runner(res.RunnerID)
	if err != nil {
		return nil, err
	}
	runnerDTO := runnerToDTO(*runner)
	isAdmin := auth.ClaimsFromContext(reqCtx) != nil
	ownsRunner := runner.AccountID != "" && runner.AccountID == auth.DSRSubject(reqCtx)

	// DNS ("did not start") is admin-only. Slightly more personal a data
	// point than DNF — public race results omit it; admins and the runner
	// themselves can still see it via their own DSR session.
	if res.Status == models.StatusDNS && !isAdmin && !ownsRunner {
		return nil, errResultRedactedAway
	}

	// Pending-deletion runners stay on public leaderboards during the grace
	// window — name forced to "Anonym", profile link dropped — so race
	// history stays coherent. (After the retention job runs, the row's
	// already persisted as name="Anonym"; this code path becomes a no-op
	// for them.)
	//
	// We deliberately do NOT honour the admin / owner bypass here. Once a
	// user has scheduled deletion, the leaderboard should reflect that
	// uniformly for every viewer — anything else is a confusing "deleted
	// but still visible to me" inconsistency. Admins can still see real
	// names via /admin/runners or /dsr/me's owner view; owners can see
	// their own data in /my-data until retention runs. The public race
	// page is the user-facing surface that respects the deletion request.
	if runner.DeletionPendingUntil != nil {
		runnerDTO.Name = "Anonym"
		runnerDTO.ID = ""
		runnerDTO.BirthYear = nil
		res.RunnerID = ""
	} else if !isAdmin && !ownsRunner {
		// Standard A0.8 redaction. Minor-age check uses the race's event
		// date so a runner who was 12 then stays redacted in that race's
		// leaderboard even after they turn 13 (and only the original
		// consent applies).
		raceDate, _ := time.Parse("2006-01-02", rctx.event.Date)
		if redacted := redactRunnerForPublic(&runnerDTO, *runner, raceDate); redacted {
			// The runner's profile link is hidden; clear the sibling
			// RunnerID field on the result row too so the frontend can't
			// reconstruct it.
			res.RunnerID = ""
		}
	}
	return &ResultExpandedDTO{
		ResultDTO: res,
		Race:      raceToDTO(rctx.race),
		Event:     eventToDTO(rctx.event),
		Runner:    runnerDTO,
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

func registerResults(api huma.API, s store.Store, authCfg auth.Config) {
	// MaybeAdmin + MaybeDSRSession attach session claims if present without
	// rejecting unauthenticated traffic. expandResult inspects the claims to
	// decide whether each row needs the A0.8 redaction.
	maybeAuthMW := huma.Middlewares{auth.MaybeAdmin(authCfg), auth.MaybeDSRSession(authCfg)}

	huma.Register(api, huma.Operation{
		OperationID: "list-results-by-race",
		Method:      "GET",
		Path:        "/races/{raceId}/results",
		Summary:     "List computed results for a race",
		Tags:        []string{"results"},
		Middlewares: maybeAuthMW,
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
			expanded, err := expandResult(ctx, r, *rc)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) || errors.Is(err, errResultRedactedAway) {
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
		Middlewares: maybeAuthMW,
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
				e, err := expandResult(ctx, r, *rc)
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
		Middlewares: maybeAuthMW,
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
				e, err := expandResult(ctx, r, *rc)
				if err != nil {
					if errors.Is(err, errResultRedactedAway) {
						return nil, huma.Error404NotFound("result not found")
					}
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
