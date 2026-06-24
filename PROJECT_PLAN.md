# `the-run` — project plan

Living document tracking frontend + backend progress. Update checkboxes as work
lands. New decisions or scope additions get appended at the bottom of the
relevant section (do not silently rewrite history — strike through and add a
follow-up so we can see what we changed our minds on).

## Goal

A platform for runners to view race results across yearly events. Iteration 1
proved the deploy pipeline (browser → CloudFront → S3, browser → API Gateway →
Lambda → Huma). From here we build the actual product, **frontend-first with
mocked data** so we can iterate on UX before committing to a backend schema.

## Data model (current)

`Event → Race → Result`, with categories as fields on a result:

- **Event** — yearly happening, e.g. "The Run 2026". Fields: `id`, `name`,
  `year`, `date`, `location`.
- **Race** — a discipline within an event, e.g. "5K Run", "5K Walk", "10K Run",
  "Kids' Race". Fields: `id`, `eventId`, `name`, `distanceMeters`, `discipline
  ('run' | 'walk' | 'kids')`.
- **Result** — a runner's outcome in a race. Fields: `id`, `raceId`, `runnerId`,
  `bib`, `finishSeconds`, `category { gender, ageGroup }`, `placementOverall`,
  `placementCategory`, `splits`, `conditions`, `notes`.
- **Runner** — `id`, `name`, `gender`, `birthYear`.

Source of truth (today): `frontend/src/lib/types.ts` + `frontend/src/lib/mock/data.ts`.

When the backend lands, types should be regenerated from Huma's OpenAPI spec to
keep frontend and backend in lockstep.

## Frontend

### Phase F1 — Sketch the runner-centric UI with mocked data

- [x] Install Skeleton v4 (Tailwind 4) + layerchart.
- [x] Wire Tailwind/Skeleton theme via `app.css`, `vite.config.ts`, `app.html`.
- [x] Add nav shell in `+layout.svelte`.
- [x] TypeScript domain types in `src/lib/types.ts`.
- [x] Mock fixtures in `src/lib/mock/data.ts` covering: 2 events, multiple
  disciplines (run/walk/kids), multiple genders/age groups, 5 runners with
  cross-year comparisons.
- [x] API stubs in `src/lib/api.ts` (`listRunners`, `getRunner`,
  `listResultsForRunner`, `getResult`, `listEvents`, `listRacesForEvent`).
- [x] Landing page (`/`) with events list, runner shortcuts, and a collapsed
  API smoke-test block.
- [x] Runners list (`/runners`) with name search.
- [x] Runner profile (`/runners/[id]`) with summary stats + sortable/filterable
  results table.
- [x] Result detail (`/results/[id]`) with splits, conditions, notes.
- [x] Charts on the runner profile (pace over time, distance distribution).
  Currently hand-rolled SVG; can be migrated to layerchart when we need
  tooltips/zoom.
- [x] i18n: Swedish default + English toggle. Lives in
  `src/lib/i18n/messages.ts` (catalog) and `src/lib/i18n/state.svelte.ts`
  (locale state, persisted to localStorage). `formatDate` is locale-aware
  (`sv-SE` / `en-GB`).

### Phase F2 — Polish + missing UX

- [x] Event detail page (`/events/[id]`) — header (name, date, location) + each
  race in the event rendered as a leaderboard (rank, runner, time, pace,
  category, bib). Reached by clicking an event card on the landing page.
- [x] Public race registration form (`/register`). Anonymous (no login), takes
  name + date of birth + gender + race. Race dropdown is restricted to races
  whose event date is today or later. CTAs from the home hero, the top nav,
  and the event detail page (upcoming events only) link in. Includes a
  honeypot field as a first-pass bot filter — see open question below.
- [ ] Real bot protection on `/register` (Cloudflare Turnstile or hCaptcha).
  Honeypot alone won't survive a determined bot; pick a provider and wire
  the token through to the backend before public launch.
- [ ] Age-group + gender filters on the results table.
- [ ] Per-runner share/print view of a single race result.
- [ ] Empty-state and error-state designs (currently minimal).
- [ ] Responsive pass: table → cards on mobile.
- [ ] Loading skeleton placeholders.

### Phase F3 — Wire to real backend

- [ ] Replace mock-data calls in `src/lib/api.ts` with `fetch` against the
  backend. Keep the same function signatures so call sites don't change.
- [ ] Generate TypeScript types from Huma OpenAPI rather than hand-maintaining
  `src/lib/types.ts`.
- [ ] Remove `src/lib/mock/data.ts` once the backend serves the same shape.

## Backend

### Phase B1 — Read-only API matching the frontend's mock surface

- [ ] DynamoDB tables for Event / Race / Result. The runner side already
  exists (`the-run-runners`, single-PK + `byNameDOB` GSI) from B-public;
  multi-table is the working assumption. Re-evaluate single-table once
  the read patterns are concrete.
- [ ] Pulumi DynamoDB table + IAM (Lambda read access).
- [ ] Huma endpoints mirroring `src/lib/api.ts`:
  - `GET /runners`
  - `GET /runners/{id}`
  - `GET /runners/{id}/results`
  - `GET /results/{id}`
  - `GET /events`
  - `GET /events/{id}/races`
- [ ] Seed script that loads the same fixture shape currently in
  `src/lib/mock/data.ts`.

### Phase B2 — Admin / back-office write API

- [ ] Auth (Cognito or equivalent — open question, see below).
- [ ] `POST` / `PUT` endpoints for events, races, results.
- [ ] CSV import for results (race-day workflow).

### Phase B-public — Public-write endpoints

- [x] `POST /registrations` — accepts `{ name, dateOfBirth, gender, raceId }`
  for the public registration form. Looks up an existing runner by
  `(name, dateOfBirth)`, creates one if missing, then writes a registration
  linking the two. Returns `{ id, runnerId, status }`. Duplicates return
  HTTP 409.
- [x] Persist runners + registrations to DynamoDB. Two tables for now —
  `the-run-runners` and `the-run-registrations`. Single-table consolidation
  is revisited when B1 (Event/Race/Result) lands.
- [ ] Validate that `raceId` exists and its event date is today or later
  (currently the frontend enforces this; backend will once Race/Event
  tables exist under B1).
- [ ] Verify the captcha token (Turnstile/hCaptcha — see F2) server-side
  before accepting the registration.

### Phase B3 — Operational concerns (added when motivated)

- [ ] Structured logging.
- [ ] Metrics + X-Ray.
- [ ] CI/CD pipeline (currently `just deploy` from a local dev shell).
- [ ] Multi-environment (dev/prod stacks).
- [ ] Integration tests against a local DynamoDB.

## Open questions

- **Chart library**: hand-rolled SVG is fine for the sketch. `layerchart` is
  installed. Switch when we need real tooltips / zoom / responsive axes, or
  drop the dep if we don't.
- **Authentication**: deferred — the read API is fine without it. Admin write
  endpoints will force this decision. Probably Cognito + JWT verified by API
  Gateway, but open.
- **Age-group definition**: currently a free-text field (`M30-39`, `F30-39`,
  `U12`). Worth replacing with structured `{minAge, maxAge}` if races have
  different bracket schemes per event.
- **Multi-year runner identity**: are runners stable across events (one record
  for "Birger Rydbäck") or fresh per registration? Current schema assumes
  stable.
- **Splits granularity**: today we store per-km. Some races publish 5K splits
  for a marathon — generalize to `{ marker: 'km' | '5k' | 'mile', index, time }`
  when the need arises.

## Conventions

- Add a checkbox under the right phase when starting work; tick it when shipped.
- When a decision overturns an earlier one, leave the original line (struck
  through) and add the new line below so the reasoning is visible.
- Phase F2 and later are placeholders — re-shape freely as we learn from F1.
- **Local-dev AWS**: LocalStack, provisioned by the same Pulumi project under
  the `local` stack (file-backed state, no AWS creds needed). See
  `just localstack-bootstrap` / `localstack-deploy`.
