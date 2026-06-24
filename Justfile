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

# Create (if needed) and configure the S3 bucket Pulumi uses for state, then
# `pulumi login` to it. Idempotent — safe to re-run. One-time per AWS account.
bootstrap-pulumi-state region='eu-north-1':
    #!/usr/bin/env bash
    set -euo pipefail
    ACCT=$(aws sts get-caller-identity --profile "$AWS_PROFILE" --query Account --output text)
    BUCKET="the-run-pulumi-state-$ACCT"
    REGION="{{region}}"

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

    pulumi login "s3://$BUCKET?region=$REGION"

# Log Pulumi in to this account's S3 state bucket. Run after
# `bootstrap-pulumi-state` (e.g. on a new machine, or after `pulumi logout`).
pulumi-login region='eu-north-1':
    #!/usr/bin/env bash
    set -euo pipefail
    ACCT=$(aws sts get-caller-identity --profile "$AWS_PROFILE" --query Account --output text)
    pulumi login "s3://the-run-pulumi-state-$ACCT?region={{region}}"

# Run any pulumi command in infra/ with SSO credentials exported. Use this for
# anything not already wrapped in a recipe (e.g. `just pulumi stack init dev`,
# `just pulumi config set the-run:domain example.com`).
pulumi *args:
    #!/usr/bin/env bash
    set -euo pipefail
    eval "$(aws configure export-credentials --profile "$AWS_PROFILE" --format env)"
    pulumi -C infra {{args}}

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
