// Package models defines the Go structs that map 1:1 to DynamoDB items.
// Persisted shape lives here — the API DTOs in internal/api/ project from
// these types but never store anything outside of them.
//
// GDPR annotations: PII-bearing fields carry a `gdpr:"..."` struct tag that
// the ROPA generator (cmd/gen-gdpr) walks via go/ast. Adding a new PII field
// without a tag will fail TestTagCoverage in CI. Run `just gen-gdpr` to
// re-render docs/gdpr/ropa.md and docs/gdpr/dpia-screening.md after any
// annotation change.
//
//go:generate go run ../../cmd/gen-gdpr -ropa ../../../docs/gdpr/ropa.md -dpia ../../../docs/gdpr/dpia-screening.md -subprocessors ../../../docs/gdpr/subprocessors.md -retention ../../../docs/gdpr/retention.md
package models
