# the-run

Platform for managing and viewing runner race results.

| Layer       | Tech                                                          |
| ----------- | ------------------------------------------------------------- |
| Backend     | Go + [Huma v2](https://huma.rocks/) on AWS Lambda             |
| Frontend    | SvelteKit (static export) on S3 + CloudFront                  |
| Edge / DNS  | API Gateway HTTP API + Route53 + ACM                          |
| IaC         | Pulumi (Go SDK), state in self-hosted S3                      |
| Build / env | Nix flake (`nix develop`)                                     |
| Region      | `eu-north-1` (CloudFront cert in `us-east-1`)                 |

This is iteration 1: hello-world plumbing. A page on `https://<DOMAIN>` fetches
`https://api.<DOMAIN>/hello` and renders the response.

## Layout

```
backend/    # Go + Huma — single GET /hello, Lambda + local HTTP server
frontend/   # SvelteKit + adapter-static
infra/      # Pulumi Go program
flake.nix   # dev shell: go, node, pnpm, pulumi, awscli, ...
```

## Tasks

Common workflows are wired into a `Justfile` at the repo root. Run `just` to
list every recipe:

```sh
just sso-login                 # aws sso login + sts get-caller-identity
just whoami                    # show the active AWS identity
just bootstrap-pulumi-state    # one-time: S3 bucket for Pulumi state + pulumi login
just pulumi-login              # pulumi login to this account's state bucket

just dev-backend         # Go API on :8080
just dev-frontend        # SvelteKit dev server on :5173

just build-lambda        # backend/dist/lambda.zip
just build-frontend prod # frontend/build/ pointing at api.<DOMAIN>
just build               # both, ready for `pulumi up`

just lint                # golangci-lint + eslint
just fix                 # gofmt + eslint --fix + prettier
just test                # go test + vitest
just check               # lint + format-check + typecheck + tests

just deploy              # build + pulumi up
just invalidate          # bust the CloudFront cache
just verify              # curl https://api.<DOMAIN>/hello
```

All recipes that touch AWS honor an `AWS_PROFILE` variable. The Justfile
defaults it to `betongsuggan-prod`; override per-invocation with
`AWS_PROFILE=other just deploy` or set it once in your shell.

The sections below explain what these recipes do under the hood — useful when
something breaks or you need to deviate from the happy path.

## First-time setup

### 1. Enter the dev shell

```sh
nix develop
```

(or `direnv allow` if you use direnv — `.envrc` is wired up.)

### 2. AWS credentials

**Preferred: IAM Identity Center (SSO).** Assumes you've enabled Identity
Center in the AWS console and assigned `AdministratorAccess` to yourself.

One-time, register the profile with `aws-cli`:

```sh
aws configure sso
# SSO start URL:    https://<your-org>.awsapps.com/start
# SSO region:       eu-north-1   (or wherever Identity Center lives)
# Profile name:     betongsuggan-prod    (matches the Justfile default)
# CLI region:       eu-north-1
```

If you pick a different profile name, point the Justfile at it by exporting
`AWS_PROFILE` in your shell (or prefixing individual commands):

```sh
export AWS_PROFILE=my-other-profile
```

Then, each session:

```sh
just sso-login    # aws sso login + sts get-caller-identity for $AWS_PROFILE
```

**Fallback: IAM user with access keys** (acceptable for a personal dev account):

1. IAM console → create a user, no console access, attach `AdministratorAccess`.
2. Create an access key (CLI use case). Store in your password manager.
3. `aws configure --profile betongsuggan-prod` → paste credentials, region `eu-north-1`.
4. `export AWS_PROFILE=betongsuggan-prod` (or whatever profile name you used).

Never commit credentials.

### 3. Bootstrap the Pulumi state bucket

Pulumi state is self-hosted in S3. The bucket has to exist before
`pulumi login`. This is a one-time step per AWS account, wrapped in a recipe:

```sh
just bootstrap-pulumi-state
```

The recipe creates `the-run-pulumi-state-<ACCOUNT_ID>` in `eu-north-1` with
versioning, SSE-S3 encryption, and public access fully blocked, then runs
`pulumi login` against it. It's idempotent — re-running is safe and will
re-assert the bucket settings.

Pick a strong passphrase and stash it in your password manager — Pulumi uses
it to encrypt secrets in stack state (replaces the role Pulumi Cloud plays
when using its managed backend).

```sh
export PULUMI_CONFIG_PASSPHRASE="<your-passphrase>"
```

### 4. Configure the stack

```sh
cd infra
pulumi stack init dev
pulumi config set the-run:domain        example.com   # ← your domain
pulumi config set the-run:apiSubdomain  api
pulumi config set the-run:siteSubdomain ""            # apex; or "www"
pulumi config set the-run:createHostedZone true
pulumi config set aws:region            eu-north-1
```

If your Route53 hosted zone already exists, set
`the-run:createHostedZone false`.

## Day-to-day: local dev

```sh
# one-time: point the SvelteKit dev server at the local API
cp frontend/.env.example frontend/.env.local   # PUBLIC_API_BASE_URL=http://localhost:8080

# terminal 1 — Go API on :8080
just dev-backend

# terminal 2 — SvelteKit on :5173
just dev-frontend
```

Open <http://localhost:5173>. The page should render `Hello from the-run @ <timestamp>`.

Huma docs at <http://localhost:8080/docs>.

## Deploy

`just deploy` builds the Lambda zip, builds the frontend wired to
`https://api.<DOMAIN>` (domain read from `pulumi config`), and runs
`pulumi -C infra up`. Use `just deploy-preview` to see the diff without
applying.

```sh
just deploy
```

### First deploy only — register your domain's nameservers

The first `pulumi up` creates the Route53 hosted zone, but ACM cert
validation will hang until your domain's registrar is pointing at the
zone's nameservers. Watch for the `hostedZoneNameServers` output and copy
those four NS values to your registrar's DNS settings. Validation will
complete automatically once DNS propagates (usually minutes, sometimes
longer).

### Subsequent deploys

After redeploying the frontend, bust CloudFront's cache:

```sh
just invalidate
```

## Verifying the deploy

```sh
just verify   # curls https://api.<DOMAIN>/hello and prints site + docs URLs
```

## What's deliberately not here yet

DynamoDB / persistence, auth, CI/CD, multiple environments,
observability (structured logging, metrics, X-Ray), tests, security
headers, custom error pages. Each gets added when there's a feature
that motivates it — see the iteration-1 plan for the rationale.
