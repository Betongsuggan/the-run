package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/audit"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// EmailTemplateDTO is the wire shape of one editable transactional email.
type EmailTemplateDTO struct {
	Slug               string   `json:"slug"`
	DisplayName        string   `json:"displayName"`
	Status             string   `json:"status"`
	Revision           int      `json:"revision"`
	SubjectSv          string   `json:"subjectSv"`
	BodySv             string   `json:"bodySv"`
	SubjectEn          string   `json:"subjectEn"`
	BodyEn             string   `json:"bodyEn"`
	AvailableVariables []string `json:"availableVariables"`
	UpdatedAt          string   `json:"updatedAt,omitempty"`
	PublishedAt        string   `json:"publishedAt,omitempty"`
}

type EmailTemplateRevisionDTO struct {
	Slug      string `json:"slug"`
	Revision  int    `json:"revision"`
	SubjectSv string `json:"subjectSv"`
	BodySv    string `json:"bodySv"`
	SubjectEn string `json:"subjectEn"`
	BodyEn    string `json:"bodyEn"`
	EditedAt  string `json:"editedAt"`
	Note      string `json:"note,omitempty"`
	Published bool   `json:"published"`
}

func emailTemplateToDTO(t models.EmailTemplate) EmailTemplateDTO {
	dto := EmailTemplateDTO{
		Slug:               string(t.Slug),
		DisplayName:        t.DisplayName,
		Status:             string(t.Status),
		Revision:           t.Revision,
		SubjectSv:          t.SubjectSv,
		BodySv:             t.BodySv,
		SubjectEn:          t.SubjectEn,
		BodyEn:             t.BodyEn,
		AvailableVariables: t.AvailableVariables,
		UpdatedAt:          t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.PublishedAt != nil {
		dto.PublishedAt = t.PublishedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func emailTemplateRevisionToDTO(r models.EmailTemplateRevision) EmailTemplateRevisionDTO {
	return EmailTemplateRevisionDTO{
		Slug:      string(r.Slug),
		Revision:  r.Revision,
		SubjectSv: r.SubjectSv,
		BodySv:    r.BodySv,
		SubjectEn: r.SubjectEn,
		BodyEn:    r.BodyEn,
		EditedAt:  r.EditedAt.UTC().Format(time.RFC3339),
		Note:      r.Note,
		Published: r.Published,
	}
}

type emailTemplateListOutput struct {
	Body struct {
		Templates []EmailTemplateDTO `json:"templates"`
	}
}

type emailTemplateSlugInput struct {
	Slug string `path:"slug"`
}

type emailTemplateGetOutput struct {
	Body EmailTemplateDTO
}

type editEmailTemplateInput struct {
	Slug string `path:"slug"`
	Body struct {
		SubjectSv string `json:"subjectSv" minLength:"1"`
		BodySv    string `json:"bodySv" minLength:"1"`
		SubjectEn string `json:"subjectEn" minLength:"1"`
		BodyEn    string `json:"bodyEn" minLength:"1"`
		Note      string `json:"note,omitempty" maxLength:"500"`
	}
}

type emailTemplateRevisionsOutput struct {
	Body struct {
		Revisions []EmailTemplateRevisionDTO `json:"revisions"`
	}
}

func registerAdminEmailTemplates(api huma.API, s store.Store, authCfg auth.Config, recorder audit.Recorder) {
	adminMW := adminMiddlewares(authCfg)

	auditTemplateAction := func(ctx context.Context, slug, action, summary, diff string) {
		actor := "admin"
		sub := auth.Subject(ctx)
		if sub != "" {
			actor = "admin:" + sub
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  sub,
			Action:     action,
			Actor:      actor,
			TargetType: "email-template",
			TargetID:   slug,
			Summary:    summary,
			Diff:       diff,
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-email-templates",
		Method:      "GET",
		Path:        "/admin/email-templates",
		Summary:     "List every editable transactional email template",
		Description: "Returns the head record for each template (current published copy + status). Use /admin/email-templates/{slug}/revisions to see the edit history.",
		Tags:        []string{"email-templates"},
		Middlewares: adminMW,
	}, func(ctx context.Context, _ *struct{}) (*emailTemplateListOutput, error) {
		templates, err := s.ListEmailTemplates(ctx)
		if err != nil {
			return nil, fmt.Errorf("list email templates: %w", err)
		}
		out := &emailTemplateListOutput{}
		out.Body.Templates = make([]EmailTemplateDTO, len(templates))
		for i, t := range templates {
			out.Body.Templates[i] = emailTemplateToDTO(t)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-email-template",
		Method:      "GET",
		Path:        "/admin/email-templates/{slug}",
		Summary:     "Get one email template by slug",
		Tags:        []string{"email-templates"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *emailTemplateSlugInput) (*emailTemplateGetOutput, error) {
		if !models.IsKnownEmailTemplateSlug(in.Slug) {
			return nil, huma.Error404NotFound("unknown template slug")
		}
		tmpl, err := s.GetEmailTemplate(ctx, models.EmailTemplateSlug(in.Slug))
		if err != nil {
			return nil, fmt.Errorf("get email template: %w", err)
		}
		if tmpl == nil {
			// Seed should populate this on first boot, but if a deploy raced
			// ahead of the seed (or LocalStack tables were destroyed), fall
			// back to the in-code defaults so the admin UI still works.
			seed, ok := email.SeededTemplate(models.EmailTemplateSlug(in.Slug))
			if !ok {
				return nil, huma.Error404NotFound("template not found")
			}
			fallback := models.EmailTemplate{
				Slug:               seed.Slug,
				DisplayName:        seed.DisplayName,
				Status:             models.EmailTemplateStatusDraft,
				Revision:           1,
				SubjectSv:          seed.SubjectSv,
				BodySv:             seed.BodySv,
				SubjectEn:          seed.SubjectEn,
				BodyEn:             seed.BodyEn,
				AvailableVariables: seed.AvailableVariables,
			}
			return &emailTemplateGetOutput{Body: emailTemplateToDTO(fallback)}, nil
		}
		return &emailTemplateGetOutput{Body: emailTemplateToDTO(*tmpl)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-edit-email-template",
		Method:      "PUT",
		Path:        "/admin/email-templates/{slug}",
		Summary:     "Edit an email template (draft body, or minor edit of a published one)",
		Description: "If the template is in draft, the bodies are simply replaced. If published, the edit bumps the revision counter and appends a snapshot row.",
		Tags:        []string{"email-templates"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *editEmailTemplateInput) (*emailTemplateGetOutput, error) {
		if !models.IsKnownEmailTemplateSlug(in.Slug) {
			return nil, huma.Error404NotFound("unknown template slug")
		}
		editor := auth.Subject(ctx)
		slug := models.EmailTemplateSlug(in.Slug)
		existing, err := s.GetEmailTemplate(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("lookup email template: %w", err)
		}
		if existing == nil {
			return nil, huma.Error404NotFound("template not found (seed has not run yet)")
		}
		switch existing.Status {
		case models.EmailTemplateStatusDraft:
			if err := s.UpdateEmailTemplateDraft(ctx, slug, in.Body.SubjectSv, in.Body.BodySv, in.Body.SubjectEn, in.Body.BodyEn, editor); err != nil {
				if errors.Is(err, store.ErrInvalidEmailTemplateState) {
					return nil, huma.Error409Conflict("template is no longer a draft")
				}
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound("template not found")
				}
				return nil, fmt.Errorf("update draft: %w", err)
			}
			auditTemplateAction(ctx, in.Slug, "admin.email-template.edited", "Admin edited draft", fmt.Sprintf("slug=%s revision=1", in.Slug))
		case models.EmailTemplateStatusPublished:
			newRev, err := s.EditPublishedEmailTemplate(ctx, slug, in.Body.SubjectSv, in.Body.BodySv, in.Body.SubjectEn, in.Body.BodyEn, editor, in.Body.Note)
			if err != nil {
				if errors.Is(err, store.ErrInvalidEmailTemplateState) {
					return nil, huma.Error409Conflict("template is no longer published")
				}
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound("template not found")
				}
				return nil, fmt.Errorf("edit published: %w", err)
			}
			summary := "Admin edited published email template"
			if note := strings.TrimSpace(in.Body.Note); note != "" {
				summary = summary + ": " + note
			}
			auditTemplateAction(ctx, in.Slug, "admin.email-template.edited", summary, fmt.Sprintf("slug=%s from-revision=%d to-revision=%d", in.Slug, existing.Revision, newRev))
		case models.EmailTemplateStatusArchived:
			return nil, huma.Error409Conflict("archived templates cannot be edited")
		default:
			return nil, huma.Error409Conflict("template is in an unknown state")
		}
		updated, err := s.GetEmailTemplate(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("re-read template: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("template not found")
		}
		return &emailTemplateGetOutput{Body: emailTemplateToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-publish-email-template",
		Method:        "POST",
		Path:          "/admin/email-templates/{slug}/publish",
		Summary:       "Publish a draft email template",
		Description:   "Flips a draft to published. Subsequent sends use the new body.",
		Tags:          []string{"email-templates"},
		DefaultStatus: http.StatusOK,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *emailTemplateSlugInput) (*emailTemplateGetOutput, error) {
		if !models.IsKnownEmailTemplateSlug(in.Slug) {
			return nil, huma.Error404NotFound("unknown template slug")
		}
		slug := models.EmailTemplateSlug(in.Slug)
		editor := auth.Subject(ctx)
		existing, err := s.GetEmailTemplate(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("lookup template: %w", err)
		}
		if existing == nil {
			return nil, huma.Error404NotFound("template not found")
		}
		if err := s.PublishEmailTemplate(ctx, slug, editor, time.Now().UTC()); err != nil {
			if errors.Is(err, store.ErrInvalidEmailTemplateState) {
				return nil, huma.Error409Conflict("only drafts can be published")
			}
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("template not found")
			}
			return nil, fmt.Errorf("publish template: %w", err)
		}
		auditTemplateAction(ctx, in.Slug, "admin.email-template.published", "Admin published email template", fmt.Sprintf("slug=%s revision=%d", in.Slug, existing.Revision))
		updated, err := s.GetEmailTemplate(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("re-read template: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("template not found")
		}
		return &emailTemplateGetOutput{Body: emailTemplateToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-archive-email-template",
		Method:        "POST",
		Path:          "/admin/email-templates/{slug}/archive",
		Summary:       "Archive a published email template",
		Description:   "Retires a template. The renderer will fall back to the in-code seed body until the template is re-published.",
		Tags:          []string{"email-templates"},
		DefaultStatus: http.StatusOK,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *emailTemplateSlugInput) (*emailTemplateGetOutput, error) {
		if !models.IsKnownEmailTemplateSlug(in.Slug) {
			return nil, huma.Error404NotFound("unknown template slug")
		}
		slug := models.EmailTemplateSlug(in.Slug)
		if err := s.ArchiveEmailTemplate(ctx, slug); err != nil {
			if errors.Is(err, store.ErrInvalidEmailTemplateState) {
				return nil, huma.Error409Conflict("only published templates can be archived")
			}
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("template not found")
			}
			return nil, fmt.Errorf("archive template: %w", err)
		}
		auditTemplateAction(ctx, in.Slug, "admin.email-template.archived", "Admin archived email template", "slug="+in.Slug)
		updated, err := s.GetEmailTemplate(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("re-read template: %w", err)
		}
		if updated == nil {
			return nil, huma.Error404NotFound("template not found")
		}
		return &emailTemplateGetOutput{Body: emailTemplateToDTO(*updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-email-template-revisions",
		Method:      "GET",
		Path:        "/admin/email-templates/{slug}/revisions",
		Summary:     "List the edit history of an email template",
		Description: "Newest first. Each row carries the body that was live at that revision.",
		Tags:        []string{"email-templates"},
		Middlewares: adminMW,
	}, func(ctx context.Context, in *emailTemplateSlugInput) (*emailTemplateRevisionsOutput, error) {
		if !models.IsKnownEmailTemplateSlug(in.Slug) {
			return nil, huma.Error404NotFound("unknown template slug")
		}
		revisions, err := s.ListEmailTemplateRevisions(ctx, models.EmailTemplateSlug(in.Slug))
		if err != nil {
			return nil, fmt.Errorf("list email template revisions: %w", err)
		}
		out := &emailTemplateRevisionsOutput{}
		out.Body.Revisions = make([]EmailTemplateRevisionDTO, len(revisions))
		for i, r := range revisions {
			out.Body.Revisions[i] = emailTemplateRevisionToDTO(r)
		}
		return out, nil
	})
}
