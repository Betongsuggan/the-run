set shell := ["bash", "-euo", "pipefail", "-c"]

# AWS profile used by aws-cli and Pulumi. Override with AWS_PROFILE=foo or
# `just aws_profile=foo <recipe>`.
export AWS_PROFILE := env_var_or_default("AWS_PROFILE", "betongsuggan-prod")

# List available recipes
default:
    @just --list --unsorted

# ─── AWS auth ─────────────────────────────────────────────────────────

# Log in to AWS IAM Identity Center for $AWS_PROFILE
sso-login:
    aws sso login --profile "$AWS_PROFILE"
    aws sts get-caller-identity --profile "$AWS_PROFILE"

# Print the active AWS identity for $AWS_PROFILE
whoami:
    @echo "AWS_PROFILE=$AWS_PROFILE"
    aws sts get-caller-identity --profile "$AWS_PROFILE"

# ─── Bootstrap ────────────────────────────────────────────────────────

# Create (if needed) the S3 state bucket, the KMS key for Pulumi stack
# secrets, `pulumi login`, and initialize the Pulumi stack. Idempotent —
# safe to re-run. One-time per AWS account (+ one-time per stack name).
bootstrap-pulumi-state stack='dev' region='eu-north-1':
    #!/usr/bin/env bash
    set -euo pipefail
    ACCT=$(aws sts get-caller-identity --profile "$AWS_PROFILE" --query Account --output text)
    BUCKET="the-run-pulumi-state-$ACCT"
    REGION="{{region}}"
    KMS_ALIAS="alias/the-run-pulumi-secrets"
    KMS_URL="awskms://$KMS_ALIAS?region=$REGION"
    STACK="{{stack}}"
    QUALIFIED_STACK="organization/the-run/$STACK"

    # --- S3 bucket for Pulumi state -----------------------------------------
    if aws s3api head-bucket --profile "$AWS_PROFILE" --bucket "$BUCKET" 2>/dev/null; then
      echo "Bucket s3://$BUCKET already exists — skipping create."
    else
      echo "Creating s3://$BUCKET in $REGION ..."
      aws s3api create-bucket --profile "$AWS_PROFILE" \
        --bucket "$BUCKET" --region "$REGION" \
        --create-bucket-configuration LocationConstraint="$REGION"
    fi

    aws s3api put-bucket-versioning --profile "$AWS_PROFILE" \
      --bucket "$BUCKET" \
      --versioning-configuration Status=Enabled
    aws s3api put-public-access-block --profile "$AWS_PROFILE" \
      --bucket "$BUCKET" \
      --public-access-block-configuration \
      "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true"
    aws s3api put-bucket-encryption --profile "$AWS_PROFILE" \
      --bucket "$BUCKET" \
      --server-side-encryption-configuration \
      '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

    # --- KMS key for Pulumi stack secrets -----------------------------------
    if aws kms describe-key --profile "$AWS_PROFILE" --region "$REGION" --key-id "$KMS_ALIAS" >/dev/null 2>&1; then
      echo "KMS key $KMS_ALIAS already exists — skipping create."
    else
      echo "Creating KMS key $KMS_ALIAS in $REGION ..."
      KEY_ID=$(aws kms create-key --profile "$AWS_PROFILE" --region "$REGION" \
        --description "Pulumi stack secrets for the-run" \
        --tags "TagKey=Project,TagValue=the-run" \
        --query 'KeyMetadata.KeyId' --output text)
      aws kms create-alias --profile "$AWS_PROFILE" --region "$REGION" \
        --alias-name "$KMS_ALIAS" --target-key-id "$KEY_ID"
      aws kms enable-key-rotation --profile "$AWS_PROFILE" --region "$REGION" --key-id "$KEY_ID"
    fi

    # --- Pulumi backend + stack ---------------------------------------------
    pulumi login "s3://$BUCKET?region=$REGION"

    # Materialize SSO creds so `pulumi stack init` can read/write S3 + KMS.
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"

    if pulumi -C infra stack select "$STACK" >/dev/null 2>&1; then
      echo "Pulumi stack $STACK already initialized — skipping."
    else
      echo "Initializing Pulumi stack $QUALIFIED_STACK with KMS secrets provider ..."
      pulumi -C infra stack init --secrets-provider="$KMS_URL" "$QUALIFIED_STACK"
    fi

# Log Pulumi in to this account's S3 state bucket. Run after
# `bootstrap-pulumi-state` (e.g. on a new machine, or after `pulumi logout`).
pulumi-login region='eu-north-1':
    #!/usr/bin/env bash
    set -euo pipefail
    ACCT=$(aws sts get-caller-identity --profile "$AWS_PROFILE" --query Account --output text)
    pulumi login "s3://the-run-pulumi-state-$ACCT?region={{region}}"

# ─── Local dev ─────────────────────────────────────────────────────────

# Run the Go API locally on :8080 against the LocalStack DynamoDB tables.
# Prerequisites (one-time): `just localstack-bootstrap`. Per-session: the
# LocalStack container must be running — `just localstack-up`.
dev-backend:
    #!/usr/bin/env bash
    set -euo pipefail
    cd backend
    export AWS_ENDPOINT_URL=http://localhost:4566
    export AWS_REGION=eu-north-1
    export AWS_ACCESS_KEY_ID=test
    export AWS_SECRET_ACCESS_KEY=test
    export RUNNERS_TABLE_NAME=the-run-runners
    export REGISTRATIONS_TABLE_NAME=the-run-registrations
    export EVENTS_TABLE_NAME=the-run-events
    export RACES_TABLE_NAME=the-run-races
    export ACCOUNTS_TABLE_NAME=the-run-accounts
    export AUTH_ATTEMPTS_TABLE_NAME=the-run-auth-attempts
    # Local dev only: a fixed JWT_SECRET avoids the Secrets Manager dependency.
    # Production reads from Secrets Manager via JWT_SECRET_ARN — see infra/backend.
    export JWT_SECRET=local-dev-only-do-not-ship-this-must-be-32-chars-or-more
    # Cookies set by the dev API must NOT be Secure (http://) and the cookie
    # domain must be unset so the browser scopes them to localhost.
    export INSECURE_COOKIES=1
    go run ./cmd/api

# Create the first admin account directly in DynamoDB (skips the auth-protected
# API). Use after a fresh deploy or to bootstrap a new environment.
# Usage:
#   just create-admin email=you@example.com password='at least 12 chars'
# Add force=1 to rotate the password of an existing account.
create-admin email='' password='' force='':
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "{{email}}" ] || [ -z "{{password}}" ]; then
      echo "usage: just create-admin email=... password=..." >&2
      exit 2
    fi
    cd backend
    if [ -n "${AWS_PROFILE:-}" ] && [ -z "${AWS_ENDPOINT_URL:-}" ]; then
      eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    fi
    : "${RUNNERS_TABLE_NAME:=the-run-runners}"
    : "${REGISTRATIONS_TABLE_NAME:=the-run-registrations}"
    : "${EVENTS_TABLE_NAME:=the-run-events}"
    : "${RACES_TABLE_NAME:=the-run-races}"
    : "${ACCOUNTS_TABLE_NAME:=the-run-accounts}"
    : "${AUTH_ATTEMPTS_TABLE_NAME:=the-run-auth-attempts}"
    export RUNNERS_TABLE_NAME REGISTRATIONS_TABLE_NAME EVENTS_TABLE_NAME RACES_TABLE_NAME ACCOUNTS_TABLE_NAME AUTH_ATTEMPTS_TABLE_NAME
    if [ -n "{{force}}" ]; then
      go run ./cmd/admin -email "{{email}}" -password "{{password}}" -force
    else
      go run ./cmd/admin -email "{{email}}" -password "{{password}}"
    fi

# Wipe all the-run-* tables in LocalStack and re-create them via Pulumi. Use
# when iterating on schema changes during development.
reset-data: localstack-up
    #!/usr/bin/env bash
    set -euo pipefail
    {{ localstack_env }}
    export PULUMI_BACKEND_URL="{{ localstack_backend }}"
    export PULUMI_CONFIG_PASSPHRASE="${PULUMI_CONFIG_PASSPHRASE:-localstack}"
    echo "Destroying LocalStack tables…"
    pulumi -C infra destroy --stack local --yes --target-dependents \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::runners-table" \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::registrations-table" \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::events-table" \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::races-table" \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::accounts-table" \
      --target "urn:pulumi:local::the-run::aws:dynamodb/table:Table::auth-attempts-table" \
      || true
    echo "Re-creating tables…"
    pulumi -C infra up --stack local --yes

# Run the SvelteKit dev server on :5173
dev-frontend:
    cd frontend && pnpm dev

# Seed the local API (must already be running on :8080 via `just dev-backend`)
# with the fixture in backend/cmd/seed/main.go. Idempotent.
seed-dev api='http://localhost:8080':
    cd backend && API_BASE_URL={{api}} go run ./cmd/seed

# ─── LocalStack ────────────────────────────────────────────────────────

# Pulumi backend URL used by the `local` stack. File-backed so we never need
# AWS creds for LocalStack workflows; isolated from the S3-backed `dev` stack.
localstack_backend := "file://" + justfile_directory() + "/infra/.pulumi-local"

# Shared env block that points every AWS SDK call at LocalStack. AWS_ENDPOINT_URL
# is honored by aws-sdk-go-v2 (which the pulumi-aws provider uses), so this one
# var covers every service the provider might touch — DynamoDB, STS, tagging,
# etc. — without us having to enumerate them per service.
localstack_env := '''
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_REGION=eu-north-1
export AWS_DEFAULT_REGION=eu-north-1
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
'''

# Start LocalStack in the background (idempotent).
localstack-up:
    localstack start -d
    localstack wait -t 60

# Stop the LocalStack container.
localstack-down:
    localstack stop

# One-time: initialize the `local` Pulumi stack using a file backend and set
# its config flags. Re-running is safe — it skips already-initialized stacks.
localstack-bootstrap:
    #!/usr/bin/env bash
    set -euo pipefail
    {{ localstack_env }}
    mkdir -p "{{ justfile_directory() }}/infra/.pulumi-local"
    export PULUMI_BACKEND_URL="{{ localstack_backend }}"
    export PULUMI_CONFIG_PASSPHRASE="${PULUMI_CONFIG_PASSPHRASE:-localstack}"
    if pulumi -C infra stack select local >/dev/null 2>&1; then
      echo "Pulumi stack local already initialized — skipping."
    else
      pulumi -C infra stack init local
    fi
    pulumi -C infra config set the-run:localstack true --stack local
    pulumi -C infra config set the-run:domain localhost --stack local

# Apply the Pulumi `local` stack to LocalStack. Starts LocalStack first.
localstack-deploy: localstack-up
    #!/usr/bin/env bash
    set -euo pipefail
    {{ localstack_env }}
    export PULUMI_BACKEND_URL="{{ localstack_backend }}"
    export PULUMI_CONFIG_PASSPHRASE="${PULUMI_CONFIG_PASSPHRASE:-localstack}"
    pulumi -C infra up --stack local --yes

# Tear down the LocalStack-deployed resources (does not stop the container).
localstack-destroy:
    #!/usr/bin/env bash
    set -euo pipefail
    {{ localstack_env }}
    export PULUMI_BACKEND_URL="{{ localstack_backend }}"
    export PULUMI_CONFIG_PASSPHRASE="${PULUMI_CONFIG_PASSPHRASE:-localstack}"
    pulumi -C infra destroy --stack local --yes

# List DynamoDB tables in LocalStack (sanity check).
localstack-tables:
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
      aws --endpoint-url=http://localhost:4566 --region=eu-north-1 \
      dynamodb list-tables

# ─── Build ────────────────────────────────────────────────────────────

# Cross-compile the Lambda binary and zip it (backend/dist/lambda.zip)
build-lambda:
    cd backend && rm -rf dist && mkdir -p dist
    cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -ldflags="-s -w" -o dist/bootstrap ./cmd/api
    cd backend/dist && zip -9q lambda.zip bootstrap

# Build the SvelteKit static site. mode = `local` (.env.local) or `prod` (https://api.<DOMAIN>)
build-frontend mode='local':
    #!/usr/bin/env bash
    set -euo pipefail
    cd frontend
    case "{{mode}}" in
      local) pnpm build ;;
      prod)
        eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
        DOMAIN=$(pulumi -C ../infra config get the-run:domain)
        PUBLIC_API_BASE_URL="https://api.${DOMAIN}" pnpm build
        ;;
      *) echo "build-frontend: mode must be 'local' or 'prod'" >&2; exit 2 ;;
    esac

# Build both for deployment (Lambda zip + frontend wired to api.<DOMAIN>)
build: build-lambda (build-frontend "prod")

# ─── Lint / format ────────────────────────────────────────────────────

# Lint all code (golangci-lint + eslint)
lint:
    cd backend  && golangci-lint run ./...
    cd infra    && golangci-lint run ./...
    cd frontend && pnpm run lint

# Auto-fix what's auto-fixable (gofmt + eslint --fix + prettier)
fix:
    gofmt -w backend infra
    cd frontend && pnpm run lint:fix
    cd frontend && pnpm run format

# Verify formatting without changing anything
format-check:
    @bad=$(gofmt -l backend infra); if [ -n "$bad" ]; then echo "gofmt: files need formatting:"; echo "$bad"; exit 1; fi
    cd frontend && pnpm run format:check

# Run svelte-check (frontend type-check)
typecheck:
    cd frontend && pnpm run check

# ─── Tests ────────────────────────────────────────────────────────────

# Run all tests (go test + vitest)
test:
    cd backend && go test ./...
    cd frontend && pnpm test

# Lint + format-check + typecheck + tests — the full CI gate
check: lint format-check typecheck test

# ─── Deploy ───────────────────────────────────────────────────────────

# Build everything and apply the Pulumi stack
deploy: build
    #!/usr/bin/env bash
    set -euo pipefail
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    pulumi -C infra up

# Build everything and show the Pulumi diff (no apply)
deploy-preview: build
    #!/usr/bin/env bash
    set -euo pipefail
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    pulumi -C infra preview

# Invalidate the entire CloudFront distribution
invalidate:
    #!/usr/bin/env bash
    set -euo pipefail
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    aws cloudfront create-invalidation --distribution-id "$(pulumi -C infra stack output cloudFrontDistributionId)" --paths '/*'

# Curl the deployed API and print the URLs
verify:
    #!/usr/bin/env bash
    set -euo pipefail
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    DOMAIN=$(pulumi -C infra config get the-run:domain)
    echo "GET https://api.${DOMAIN}/hello"
    curl -fsSL "https://api.${DOMAIN}/hello" | jq .
    echo
    echo "Site:     https://${DOMAIN}"
    echo "API docs: https://api.${DOMAIN}/docs"

# ─── Cleanup ──────────────────────────────────────────────────────────

# Remove build artifacts
clean:
    rm -rf backend/dist backend/bootstrap
    rm -rf frontend/build frontend/.svelte-kit frontend/coverage
