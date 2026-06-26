package models

import "time"

// PolicyStatus is the lifecycle of a privacy-policy record.
//
//   - Draft     — admin authoring; not visible to runners and not used for
//                 consent stamping.
//   - Published — exactly one policy is published at a time. New consents
//                 reference this one's ID + revision.
//   - Archived  — was once published; retained so old consent records still
//                 resolve back to the body the user accepted.
type PolicyStatus string

const (
	PolicyStatusDraft     PolicyStatus = "draft"
	PolicyStatusPublished PolicyStatus = "published"
	PolicyStatusArchived  PolicyStatus = "archived"
)

// Policy is the current head of a single privacy-policy record. Body is
// markdown in both Swedish and English; rendering happens client-side after
// DOMPurify sanitization. Revision bumps every time an admin makes a minor
// edit to a published policy — see PolicyRevision for the append-only history.
type Policy struct {
	ID            string
	Slug          string
	Status        PolicyStatus
	Revision      int
	EffectiveFrom time.Time
	BodySv        string
	BodyEn        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PublishedAt   *time.Time
	PublishedBy   string
	LastEditedBy  string
}

// PolicyRevision is one row in the append-only history for a policy. The
// initial draft creates revision 1; each in-place edit appends a new
// revision row while also updating the head body in the Policy record.
// Consents reference (PolicyID, Revision) so the exact accepted text can
// always be retrieved.
type PolicyRevision struct {
	PolicyID  string
	Revision  int
	BodySv    string
	BodyEn    string
	EditedAt  time.Time
	EditedBy  string
	Note      string
	Published bool
}
