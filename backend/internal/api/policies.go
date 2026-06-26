package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// PolicyDTO is the public projection of a privacy policy — both the
// markdown bodies and the metadata callers need to render a versioned
// consent UI.
type PolicyDTO struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Status        string `json:"status"`
	Revision      int    `json:"revision"`
	EffectiveFrom string `json:"effectiveFrom"`
	BodySv        string `json:"bodySv"`
	BodyEn        string `json:"bodyEn"`
	PublishedAt   string `json:"publishedAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// PolicyRevisionDTO is one historical snapshot of a policy body. Used by
// /my-data to render the exact text a runner consented to, even after the
// admin made minor edits to the head.
type PolicyRevisionDTO struct {
	PolicyID  string `json:"policyId"`
	Revision  int    `json:"revision"`
	BodySv    string `json:"bodySv"`
	BodyEn    string `json:"bodyEn"`
	EditedAt  string `json:"editedAt"`
	Note      string `json:"note,omitempty"`
	Published bool   `json:"published"`
}

func policyToDTO(p models.Policy) PolicyDTO {
	dto := PolicyDTO{
		ID:            p.ID,
		Slug:          p.Slug,
		Status:        string(p.Status),
		Revision:      p.Revision,
		EffectiveFrom: p.EffectiveFrom.UTC().Format(time.RFC3339),
		BodySv:        p.BodySv,
		BodyEn:        p.BodyEn,
		UpdatedAt:     p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.PublishedAt != nil {
		dto.PublishedAt = p.PublishedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func policyRevisionToDTO(r models.PolicyRevision) PolicyRevisionDTO {
	return PolicyRevisionDTO{
		PolicyID:  r.PolicyID,
		Revision:  r.Revision,
		BodySv:    r.BodySv,
		BodyEn:    r.BodyEn,
		EditedAt:  r.EditedAt.UTC().Format(time.RFC3339),
		Note:      r.Note,
		Published: r.Published,
	}
}

type currentPolicyOutput struct {
	Body PolicyDTO
}

type policyRevisionInput struct {
	PolicyID string `path:"policyId"`
	Revision int    `path:"revision"`
}

type policyRevisionOutput struct {
	Body PolicyRevisionDTO
}

func registerPolicies(api huma.API, s store.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-current-policy",
		Method:      "GET",
		Path:        "/policies/current",
		Summary:     "Get the currently-published privacy policy",
		Description: "Returns the policy that new consents are stamped against. 503 if none is published yet — registrations are paused until an admin publishes one.",
		Tags:        []string{"policies"},
	}, func(ctx context.Context, _ *struct{}) (*currentPolicyOutput, error) {
		p, err := s.GetPublishedPolicy(ctx)
		if err != nil {
			if errors.Is(err, store.ErrNoPublishedPolicy) {
				return nil, huma.Error503ServiceUnavailable("no privacy policy is published")
			}
			return nil, fmt.Errorf("get published policy: %w", err)
		}
		return &currentPolicyOutput{Body: policyToDTO(*p)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-policy-revision",
		Method:      "GET",
		Path:        "/policies/{policyId}/revisions/{revision}",
		Summary:     "Get a historical revision of a policy",
		Description: "Returns the exact body that was live at the named revision. Used by /my-data so a runner can read what they consented to even after admin edits.",
		Tags:        []string{"policies"},
	}, func(ctx context.Context, in *policyRevisionInput) (*policyRevisionOutput, error) {
		rev, err := s.GetPolicyRevision(ctx, in.PolicyID, in.Revision)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("policy revision not found")
			}
			return nil, fmt.Errorf("get policy revision: %w", err)
		}
		return &policyRevisionOutput{Body: policyRevisionToDTO(*rev)}, nil
	})
}

// ─── Admin endpoints ──────────────────────────────────────────────────

type adminPolicyListOutput struct {
	Body struct {
		Policies []PolicyDTO `json:"policies"`
	}
}

type createPolicyInput struct {
	Body struct {
		Slug          string `json:"slug" minLength:"1" maxLength:"80"`
		EffectiveFrom string `json:"effectiveFrom" format:"date-time"`
		BodySv        string `json:"bodySv" minLength:"1"`
		BodyEn        string `json:"bodyEn" minLength:"1"`
	}
}

type policyIDInput struct {
	PolicyID string `path:"policyId"`
}

type editPolicyInput struct {
	PolicyID string `path:"policyId"`
	Body     struct {
		BodySv string `json:"bodySv" minLength:"1"`
		BodyEn string `json:"bodyEn" minLength:"1"`
		Note   string `json:"note,omitempty" maxLength:"500"`
	}
}

type policyOutput struct {
	Body PolicyDTO
}

type policyRevisionsOutput struct {
	Body struct {
		Revisions []PolicyRevisionDTO `json:"revisions"`
	}
}
