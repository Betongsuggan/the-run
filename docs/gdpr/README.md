# GDPR documents

This directory holds the operator-facing GDPR artefacts. Most files are
**auto-generated** from the code state; one is hand-authored.

## Files

| File | Generated? | Source |
|---|---|---|
| `ropa.md` | Yes | `gdpr:` struct tags in `backend/internal/models/` + registries in `backend/internal/gdpr/` |
| `subprocessors.md` | Yes | `backend/internal/gdpr/subprocessors.go` |
| `retention.md` | Yes | `backend/internal/gdpr/retention.go` + `storage.go` |
| `dpia-screening.md` | Mostly | Same source; the operator hand-fills prose between `<!-- gdpr:prose:NAME -->` markers |
| `breach-runbook.md` | No | Hand-authored operational runbook |
| `dsr-process.md` | No | Hand-authored DSR fulfilment process |
| `cookie-analysis.md` | No | Hand-authored Art. 5(3) ePrivacy analysis (defends the no-banner decision) |
| `policy/2026-06-26-sv.md` | No | Hand-authored. Seeded into the runtime via `just seed-policy`. |
| `policy/2026-06-26-en.md` | No | Hand-authored English mirror of the same policy. |

## Re-generating

After editing any GDPR-relevant code — struct tag on a model field, a
purpose description, a retention constant, sub-processor, or table posture —
re-render the docs:

```
just gen-gdpr
```

The DPIA scaffold preserves your prose between the `gdpr:prose:NAME`
markers, so adding a new struct field doesn't blow away your risk-analysis
text.

## CI drift gate

`backend/internal/gdpr/drift_test.go` re-runs the generator into a tempdir
and compares it byte-for-byte against the committed files. A PR that adds
a tagged field without committing the regenerated ROPA fails the build with
an actionable message ("ropa.md is stale. Run `just gen-gdpr` to refresh.").

Two sibling tests guard related invariants:

- `tag_coverage_test.go` — fails if a struct field has a PII-shaped name
  (`Email`, `Name`, `BirthDate`, …) but no `gdpr:` tag. False positives
  go in the `knownNonPII` allowlist with a one-liner justification.
- `storage_consistency_test.go` — fails if the `gdpr.Tables` registry in
  `internal/gdpr/storage.go` disagrees with `infra/database/database.go`
  on SSE / PITR / TTL for any table.

## What lives where

| Concept | Source of truth |
|---|---|
| PII categorisation per field | `gdpr:` struct tag on `internal/models/*.go` |
| Purpose descriptions + lawful basis | `internal/gdpr/purposes.go` |
| Retention windows | `internal/gdpr/retention.go` (imported by `cmd/retention/`, `internal/api/`, `internal/auth/`) |
| Sub-processors | `internal/gdpr/subprocessors.go` |
| Table storage posture (SSE / PITR / TTL) | `internal/gdpr/storage.go` (cross-checked against `infra/database/database.go`) |
| Tech + org security measures | `internal/gdpr/storage.go` (`SecurityMeasures` slice) |
| Currently-published privacy policy | DynamoDB (`the-run-policies` table); seed file at `policy/2026-06-26-{sv,en}.md` |

## Adding a new PII field

1. Add the field to the struct in `internal/models/`.
2. Tag it: `gdpr:"<category>;purposes=<key>[,<key>]*[;subject=<override>]"`.
   See `internal/gdpr/purposes.go` for available category + purpose keys.
3. Run `just gen-gdpr`.
4. Review the diff in `docs/gdpr/ropa.md` and commit both the model + the
   regenerated doc.

If the new field is for a new processing activity, add a purpose to
`internal/gdpr/purposes.go` first.

## Adding a new sub-processor

Editing `internal/gdpr/subprocessors.go` and running `just gen-gdpr` lands
it in the ROPA's "Underbiträden" table. Make sure the privacy policy at
`policy/<slug>-{sv,en}.md` mentions them too, then re-seed / publish a
new policy version via `/admin/policies`.
