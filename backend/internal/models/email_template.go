package models

import "time"

// EmailTemplateStatus is the lifecycle of an email-template record. Mirrors
// PolicyStatus: drafts are admin-authored but not used for sending; a single
// published record per slug is what the renderer actually substitutes against.
type EmailTemplateStatus string

const (
	EmailTemplateStatusDraft     EmailTemplateStatus = "draft"
	EmailTemplateStatusPublished EmailTemplateStatus = "published"
	EmailTemplateStatusArchived  EmailTemplateStatus = "archived"
)

// EmailTemplateSlug discriminates which send site a template feeds. The set is
// closed: each slug corresponds to exactly one call site in the backend, so
// admins can edit copy but cannot create new templates without a code change.
type EmailTemplateSlug string

const (
	EmailTemplateSlugRegistrationConfirmation EmailTemplateSlug = "registration-confirmation"
	EmailTemplateSlugGuardianConsent          EmailTemplateSlug = "guardian-consent"
	EmailTemplateSlugDSRAccess                EmailTemplateSlug = "dsr-access"
	EmailTemplateSlugDSREmailChange           EmailTemplateSlug = "dsr-email-change"
	EmailTemplateSlugDSRRestore               EmailTemplateSlug = "dsr-restore"
	EmailTemplateSlugRetentionInactivity      EmailTemplateSlug = "retention-inactivity"
	EmailTemplateSlugAdminInvite              EmailTemplateSlug = "admin-invite"
)

// KnownEmailTemplateSlugs is the canonical, in-code-order list. Used by the
// seed routine, by admin endpoints that enumerate templates, and by validation
// that an unknown slug never reaches the renderer.
var KnownEmailTemplateSlugs = []EmailTemplateSlug{
	EmailTemplateSlugRegistrationConfirmation,
	EmailTemplateSlugGuardianConsent,
	EmailTemplateSlugDSRAccess,
	EmailTemplateSlugDSREmailChange,
	EmailTemplateSlugDSRRestore,
	EmailTemplateSlugRetentionInactivity,
	EmailTemplateSlugAdminInvite,
}

// IsKnownEmailTemplateSlug reports whether s is one of the slugs we ship
// handlers for.
func IsKnownEmailTemplateSlug(s string) bool {
	for _, k := range KnownEmailTemplateSlugs {
		if string(k) == s {
			return true
		}
	}
	return false
}

// EmailTemplate is the current head of a single template record. Subject and
// body are plain text in both Swedish and English; the renderer picks based on
// the recipient's account.Locale (falls back to sv).
//
// AvailableVariables documents which `{{.Field}}` placeholders the renderer
// substitutes for this template — used by the admin UI to show a reference
// panel next to the editor. The list is informational; the actual variables a
// send site passes are defined in code.
type EmailTemplate struct {
	Slug               EmailTemplateSlug
	DisplayName        string
	Status             EmailTemplateStatus
	Revision           int
	SubjectSv          string
	BodySv             string
	SubjectEn          string
	BodyEn             string
	AvailableVariables []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	PublishedAt        *time.Time
	PublishedBy        string
	LastEditedBy       string
}

// EmailTemplateRevision is one row in the append-only history for a template.
// Same shape as PolicyRevision.
type EmailTemplateRevision struct {
	Slug      EmailTemplateSlug
	Revision  int
	SubjectSv string
	BodySv    string
	SubjectEn string
	BodyEn    string
	EditedAt  time.Time
	EditedBy  string
	Note      string
	Published bool
}
