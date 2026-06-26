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

func registerAdminPolicies(api huma.API, s store.Store, authCfg auth.Config, recorder audit.Recorder) {
	adminMW := adminMiddlewares(authCfg)

	// auditPolicyAction records a policy-related action against the
	// publishing admin. TargetType: "policy", TargetID: policyId. Best-
	// effort — failures don't block the operation.
	auditPolicyAction := func(ctx context.Context, policyID, action, summary, diff string) {
		actor := "admin"
		sub := auth.Subject(ctx)
		if sub != "" {
			actor = "admin:" + sub
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  sub,
			Action:     action,
			Actor:      actor,
			TargetType: "policy",
			TargetID:   policyID,
			Summary:    summary,
			Diff:       diff,
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-policies",
		Method:      "GET",
		Path:        "/admin/policies",
		Summary:     "List all privacy policies",
		Description: "Admin view of every policy (draft, published, archived). Sorted newest first.",
		Tags:        []string{"policies"},
		Middlewares: adminMW,
	}, func(ctx context.Context, _ *struct{}) (*adminPolicyListOutput, error) {
		policies, err := s.ListPolicies(ctx)
		if err != nil {
			return nil, fmt.Errorf("list policies: %w", err)
		}
		out := &adminPolicyListOutput{}
		out.Body.Policies = make([]PolicyDTO, len(policies))
		for i, p := range policies {
			out.Body.Policies[i] = policyToDTO(p)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-create-policy",
		Method:        "POST",
		Path:          "/admin/policies",
		Summary:       "Create a new privacy-policy draft",
		Description:   "Drafts are not visible to runners. Publish a draft to make it the active policy stamped on new consents.",
		Tags:          []string{"policies"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *createPolicyInput) (*policyOutput, error) {
		slug := strings.TrimSpace(in.Body.Slug)
		if slug == "" {
			return nil, huma.Error422UnprocessableEntity("slug is required")
		}
		effectiveFrom, err := time.Parse(time.RFC3339, in.Body.EffectiveFrom)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("effectiveFrom must be RFC3339")
		}
		now := time.Now().UTC()
		editor := auth.Subject(ctx)
		p := models.Policy{
			ID:            uuid.NewString(),
			Slug:          slug,
			Status:        models.PolicyStatusDraft,
			Revision:      1,
			EffectiveFrom: effectiveFrom,
			BodySv:        in.Body.BodySv,
			BodyEn:        in.Body.BodyEn,
			CreatedAt:     now,
			UpdatedAt:     now,
			LastEditedBy:  editor,
		}
		if err := s.CreatePolicyDraft(ctx, p); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return nil, huma.Error409Conflict("a policy with that slug already exists")
			}
			return nil, fmt.Errorf("create policy draft: %w", err)
		}
		auditPolicyAction(ctx, p.ID, "admin.policy.created", "Admin created policy draft", fmt.Sprintf("slug=%s revision=1", p.Slug))
		return &policyOutput{Body: policyToDTO(p)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-edit-policy",
		Method:      "PUT",
		Path:        "/admin/policies/{policyId}",
		Summary:     "Edit a policy (draft body, or minor edit of a published one)",
		Description: "If the policy is in draft, the bodies are simply replaced. If published, the edit bumps the revision counter and appends a snapshot row to the history table.",
		Tags:        []string{"policies"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *editPolicyInput) (*policyOutput, error) {
		editor := auth.Subject(ctx)
		existing, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("lookup policy: %w", err)
		}
		if existing == nil {
			return nil, huma.Error404NotFound("policy not found")
		}
		switch existing.Status {
		case models.PolicyStatusDraft:
			if err := s.UpdatePolicyDraft(ctx, in.PolicyID, in.Body.BodySv, in.Body.BodyEn, editor); err != nil {
				if errors.Is(err, store.ErrInvalidPolicyState) {
					return nil, huma.Error409Conflict("policy is no longer a draft")
				}
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound("policy not found")
				}
				return nil, fmt.Errorf("update draft: %w", err)
			}
			auditPolicyAction(ctx, in.PolicyID, "admin.policy.edited", "Admin edited draft", fmt.Sprintf("slug=%s revision=1", existing.Slug))
		case models.PolicyStatusPublished:
			newRev, err := s.EditPublishedPolicy(ctx, in.PolicyID, in.Body.BodySv, in.Body.BodyEn, editor, in.Body.Note)
			if err != nil {
				if errors.Is(err, store.ErrInvalidPolicyState) {
					return nil, huma.Error409Conflict("policy is no longer published")
				}
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound("policy not found")
				}
				return nil, fmt.Errorf("edit published: %w", err)
			}
			diff := fmt.Sprintf("slug=%s from-revision=%d to-revision=%d", existing.Slug, existing.Revision, newRev)
			summary := "Admin edited published policy"
			if note := strings.TrimSpace(in.Body.Note); note != "" {
				summary = summary + ": " + note
			}
			auditPolicyAction(ctx, in.PolicyID, "admin.policy.edited", summary, diff)
		case models.PolicyStatusArchived:
			return nil, huma.Error409Conflict("archived policies cannot be edited")
		default:
			return nil, huma.Error409Conflict("policy is in an unknown state")
		}
		updated, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("re-read policy: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("policy not found")
		}
		return &policyOutput{Body: policyToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-publish-policy",
		Method:      "POST",
		Path:        "/admin/policies/{policyId}/publish",
		Summary:     "Publish a draft policy",
		Description: "Flips a draft to published. Atomically archives whatever was previously published so exactly one policy is active at a time.",
		Tags:        []string{"policies"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *policyIDInput) (*policyOutput, error) {
		editor := auth.Subject(ctx)
		existing, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("lookup policy: %w", err)
		}
		if existing == nil {
			return nil, huma.Error404NotFound("policy not found")
		}

		previouslyPublished, err := s.GetPublishedPolicy(ctx)
		if err != nil && !errors.Is(err, store.ErrNoPublishedPolicy) {
			return nil, fmt.Errorf("lookup current published: %w", err)
		}

		if err := s.PublishPolicy(ctx, in.PolicyID, editor, time.Now().UTC()); err != nil {
			if errors.Is(err, store.ErrInvalidPolicyState) {
				return nil, huma.Error409Conflict("only drafts can be published")
			}
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("policy not found")
			}
			return nil, fmt.Errorf("publish policy: %w", err)
		}
		auditPolicyAction(ctx, in.PolicyID, "admin.policy.published", "Admin published policy", fmt.Sprintf("slug=%s revision=%d", existing.Slug, existing.Revision))
		if previouslyPublished != nil && previouslyPublished.ID != in.PolicyID {
			auditPolicyAction(ctx, previouslyPublished.ID, "admin.policy.archived",
				"Auto-archived by publish of "+existing.Slug,
				fmt.Sprintf("replaced-by=%s", existing.Slug))
		}
		updated, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("re-read policy: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("policy not found")
		}
		return &policyOutput{Body: policyToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-archive-policy",
		Method:      "POST",
		Path:        "/admin/policies/{policyId}/archive",
		Summary:     "Archive a published policy",
		Description: "Retires a published policy without publishing a replacement. After this returns there is no current policy and /policies/current will 503 until a draft is published.",
		Tags:        []string{"policies"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *policyIDInput) (*policyOutput, error) {
		existing, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("lookup policy: %w", err)
		}
		if existing == nil {
			return nil, huma.Error404NotFound("policy not found")
		}
		if err := s.ArchivePolicy(ctx, in.PolicyID); err != nil {
			if errors.Is(err, store.ErrInvalidPolicyState) {
				return nil, huma.Error409Conflict("only published policies can be archived")
			}
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("policy not found")
			}
			return nil, fmt.Errorf("archive policy: %w", err)
		}
		auditPolicyAction(ctx, in.PolicyID, "admin.policy.archived", "Admin archived policy", fmt.Sprintf("slug=%s", existing.Slug))
		updated, err := s.GetPolicyByID(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("re-read policy: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("policy not found")
		}
		return &policyOutput{Body: policyToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-policy-revisions",
		Method:      "GET",
		Path:        "/admin/policies/{policyId}/revisions",
		Summary:     "List the edit history of a policy",
		Description: "Newest first. Each row carries the body that was live at that revision.",
		Tags:        []string{"policies"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *policyIDInput) (*policyRevisionsOutput, error) {
		revisions, err := s.ListPolicyRevisions(ctx, in.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("list policy revisions: %w", err)
		}
		out := &policyRevisionsOutput{}
		out.Body.Revisions = make([]PolicyRevisionDTO, len(revisions))
		for i, r := range revisions {
			out.Body.Revisions[i] = policyRevisionToDTO(r)
		}
		return out, nil
	})
}
