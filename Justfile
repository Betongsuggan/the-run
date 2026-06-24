set shell := ["bash", "-euo", "pipefail", "-c"]

# List available recipes
default:
    @just --list --unsorted

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
    pulumi -C infra up

# Build everything and show the Pulumi diff (no apply)
deploy-preview: build
    pulumi -C infra preview

# Invalidate the entire CloudFront distribution
invalidate:
    aws cloudfront create-invalidation --distribution-id "$(pulumi -C infra stack output cloudFrontDistributionId)" --paths '/*'

# Curl the deployed API and print the URLs
verify:
    #!/usr/bin/env bash
    set -euo pipefail
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
