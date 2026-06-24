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

# Run the Go API locally on :8080
dev-backend:
    cd backend && go run ./cmd/api

# Run the SvelteKit dev server on :5173
dev-frontend:
    cd frontend && pnpm dev

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
