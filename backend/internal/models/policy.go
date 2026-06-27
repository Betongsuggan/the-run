package models

import "time"

// PolicyStatus is the lifecycle of a policy record.
//
//   - Draft     — admin authoring; not visible to users and not used for
//                 consent stamping.
//   - Published — exactly one policy is published at a time per Kind. New
//                 consents reference this one's ID + revision.
//   - Archived  — was once published; retained so old consent records still
//                 resolve back to the body the user accepted.
type PolicyStatus string

const (
	PolicyStatusDraft     PolicyStatus = "draft"
	PolicyStatusPublished PolicyStatus = "published"
	PolicyStatusArchived  PolicyStatus = "archived"
)

// PolicyKind discriminates the user-facing policy documents we publish.
// Today only the privacy policy exists; new kinds (ToS, Code of Conduct,
// photographer release, etc.) can be added by appending to this enum, the
// admin-create Huma enum tag, and the i18n kind-label catalog.
//
// Legacy rows in DynamoDB that predate this field deserialize with the empty
// string and are normalised to PolicyKindPrivacy by the store layer — see
// policyFromItem.
type PolicyKind string

const (
	PolicyKindPrivacy PolicyKind = "privacy"
)

// DefaultPolicyKind is what the store returns when reading a row with no
// kind attribute set (back-compat with the single seeded draft that predates
// this field).
const DefaultPolicyKind = PolicyKindPrivacy

// Policy is the current head of a single policy-document record. Body is
// markdown in both Swedish and English; rendering happens client-side after
// DOMPurify sanitization. Revision bumps every time an admin makes a minor
// edit to a published policy — see PolicyRevision for the append-only history.
//
// Kind groups policy records by what user-facing document they represent
// (privacy, ToS, etc.). The "exactly one published" invariant is scoped per
// kind, so a privacy policy and a future ToS can both be published at the
// same time without colliding.
type Policy struct {
	ID            string
	Kind          PolicyKind
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
