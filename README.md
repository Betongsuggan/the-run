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

## First-time setup

### 1. Enter the dev shell

```sh
nix develop
```

(or `direnv allow` if you use direnv — `.envrc` is wired up.)

### 2. AWS credentials

**Preferred: IAM Identity Center (SSO).** Assumes you've enabled Identity
Center in the AWS console and assigned `AdministratorAccess` to yourself.

```sh
aws configure sso
# SSO start URL:    https://<your-org>.awsapps.com/start
# SSO region:       eu-north-1   (or wherever Identity Center lives)
# Profile name:     the-run
# CLI region:       eu-north-1

aws sso login --profile the-run
export AWS_PROFILE=the-run
aws sts get-caller-identity   # must succeed before continuing
```

**Fallback: IAM user with access keys** (acceptable for a personal dev account):

1. IAM console → create user `the-run-dev`, no console access, attach
   `AdministratorAccess`.
2. Create an access key (CLI use case). Store in your password manager.
3. `aws configure --profile the-run` → paste credentials, region `eu-north-1`.
4. `export AWS_PROFILE=the-run`.

Never commit credentials.

### 3. Bootstrap the Pulumi state bucket

Pulumi state is self-hosted in S3. The bucket has to exist before
`pulumi login`. This is a one-time step per AWS account.

```sh
ACCT=$(aws sts get-caller-identity --query Account --output text)
REGION=eu-north-1
BUCKET="the-run-pulumi-state-$ACCT"

aws s3api create-bucket --bucket "$BUCKET" --region "$REGION" \
  --create-bucket-configuration LocationConstraint="$REGION"
aws s3api put-bucket-versioning --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled
aws s3api put-public-access-block --bucket "$BUCKET" \
  --public-access-block-configuration \
  "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicAccess=true"
aws s3api put-bucket-encryption --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

pulumi login "s3://$BUCKET?region=$REGION"
```

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
# terminal 1 — backend on :8080
cd backend
go run ./cmd/api

# terminal 2 — frontend on :5173 pointing at local backend
cd frontend
cp .env.example .env.local      # PUBLIC_API_BASE_URL=http://localhost:8080
pnpm dev
```

Open <http://localhost:5173>. The page should render `Hello from the-run @ <timestamp>`.

Huma docs at <http://localhost:8080/docs>.

## Deploy

Order matters: Lambda zip and frontend build have to exist before `pulumi up`.

```sh
# 1. Build the Lambda zip.
cd backend
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags lambda.norpc -ldflags="-s -w" -o dist/bootstrap ./cmd/api
( cd dist && zip -9 lambda.zip bootstrap )

# 2. Build the frontend with the production API URL.
cd ../frontend
DOMAIN=$(cd ../infra && pulumi config get the-run:domain)
PUBLIC_API_BASE_URL="https://api.$DOMAIN" pnpm build

# 3. Deploy.
cd ../infra
pulumi up
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
aws cloudfront create-invalidation \
  --distribution-id "$(pulumi -C infra stack output cloudFrontDistributionId)" \
  --paths '/*'
```

## Verifying the deploy

```sh
DOMAIN=$(pulumi -C infra config get the-run:domain)

curl -s "https://api.$DOMAIN/hello" | jq .
# {"message":"Hello from the-run","timestamp":"..."}

open "https://api.$DOMAIN/docs"     # Huma's OpenAPI UI
open "https://$DOMAIN"               # the SvelteKit page
```

## What's deliberately not here yet

DynamoDB / persistence, auth, CI/CD, multiple environments,
observability (structured logging, metrics, X-Ray), tests, security
headers, custom error pages. Each gets added when there's a feature
that motivates it — see the iteration-1 plan for the rationale.
