// Package database provisions the DynamoDB tables backing the API. The same
// Setup is invoked by both the production AWS stack and the LocalStack-targeted
// local stack — `provider` is the difference (pass nil for the default
// provider).
//
// SSE is set explicitly on every table (GDPR A0.6, self-documenting even
// though it matches the AWS-managed default). PITR is enabled on every PII
// table: accounts (email + password hash), runners (name + DOB), and
// registrations (links runner to race, reconstructible into participation
// history). Domain tables (events, races) and TTL-driven tables (auth-attempts)
// skip PITR — there's nothing to recover.
package database

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// sseEnabled is the explicit AWS-managed encryption-at-rest opt-in. Applied
// to every table so a future audit reads "encryption: on" without inferring
// it from AWS defaults.
func sseEnabled() *dynamodb.TableServerSideEncryptionArgs {
	return &dynamodb.TableServerSideEncryptionArgs{Enabled: pulumi.Bool(true)}
}

func pitrEnabled() *dynamodb.TablePointInTimeRecoveryArgs {
	return &dynamodb.TablePointInTimeRecoveryArgs{Enabled: pulumi.Bool(true)}
}

const (
	RunnersTableName       = "the-run-runners"
	RegistrationsTableName = "the-run-registrations"
	EventsTableName        = "the-run-events"
	RacesTableName         = "the-run-races"
	AccountsTableName      = "the-run-accounts"
	AuthAttemptsTableName  = "the-run-auth-attempts"
	MagicTokensTableName   = "the-run-magic-tokens"
	AuditTableName         = "the-run-audit"
	RateLimitTableName     = "the-run-rate-limit"
)

type Tables struct {
	Runners       *dynamodb.Table
	Registrations *dynamodb.Table
	Events        *dynamodb.Table
	Races         *dynamodb.Table
	Accounts      *dynamodb.Table
	AuthAttempts  *dynamodb.Table
	MagicTokens   *dynamodb.Table
	Audit         *dynamodb.Table
	RateLimit     *dynamodb.Table
}

func Setup(ctx *pulumi.Context, provider *aws.Provider) (*Tables, error) {
	var opts []pulumi.ResourceOption
	if provider != nil {
		opts = append(opts, pulumi.Provider(provider))
	}

	// Runners: PK=id (S). Two GSIs:
	//  - byNameDOB: existing — same-person-re-registering dedup. With the
	//    Account model this is no longer globally unique; callers filter by
	//    accountId.
	//  - byAccount: lists all runners owned by an Account (DSR /my-data).
	runners, err := dynamodb.NewTable(ctx, "runners-table", &dynamodb.TableArgs{
		Name:        pulumi.String(RunnersTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("nameDobKey"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("accountId"), Type: pulumi.String("S")},
		},
		GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
			&dynamodb.TableGlobalSecondaryIndexArgs{
				Name:           pulumi.String("byNameDOB"),
				HashKey:        pulumi.String("nameDobKey"),
				ProjectionType: pulumi.String("ALL"),
			},
			&dynamodb.TableGlobalSecondaryIndexArgs{
				Name:           pulumi.String("byAccount"),
				HashKey:        pulumi.String("accountId"),
				ProjectionType: pulumi.String("ALL"),
			},
		},
		ServerSideEncryption: sseEnabled(),
		PointInTimeRecovery:  pitrEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	registrations, err := dynamodb.NewTable(ctx, "registrations-table", &dynamodb.TableArgs{
		Name:        pulumi.String(RegistrationsTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("raceId"),
		RangeKey:    pulumi.String("runnerId"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("raceId"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("runnerId"), Type: pulumi.String("S")},
		},
		GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
			&dynamodb.TableGlobalSecondaryIndexArgs{
				Name:           pulumi.String("byRunner"),
				HashKey:        pulumi.String("runnerId"),
				RangeKey:       pulumi.String("raceId"),
				ProjectionType: pulumi.String("ALL"),
			},
		},
		ServerSideEncryption: sseEnabled(),
		PointInTimeRecovery:  pitrEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	events, err := dynamodb.NewTable(ctx, "events-table", &dynamodb.TableArgs{
		Name:        pulumi.String(EventsTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	races, err := dynamodb.NewTable(ctx, "races-table", &dynamodb.TableArgs{
		Name:        pulumi.String(RacesTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("eventId"), Type: pulumi.String("S")},
		},
		GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
			&dynamodb.TableGlobalSecondaryIndexArgs{
				Name:           pulumi.String("byEvent"),
				HashKey:        pulumi.String("eventId"),
				ProjectionType: pulumi.String("ALL"),
			},
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Accounts table — PK=id (S). The same table holds both primary account
	// rows (id = UUID) and email-uniqueness sentinel rows (id =
	// "EMAIL#<normalized>"); they're disambiguated by the `kind` attribute.
	// PITR enabled because this is the only place email + password hash live.
	accounts, err := dynamodb.NewTable(ctx, "accounts-table", &dynamodb.TableArgs{
		Name:        pulumi.String(AccountsTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
		},
		ServerSideEncryption: sseEnabled(),
		PointInTimeRecovery:  pitrEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Auth attempts table — PK=accountId (S), SK=attemptedAt (S). TTL on
	// `expiresAt` (epoch seconds) so DynamoDB auto-deletes old failures.
	// No PITR — short-lived security telemetry, not PII to be preserved.
	authAttempts, err := dynamodb.NewTable(ctx, "auth-attempts-table", &dynamodb.TableArgs{
		Name:        pulumi.String(AuthAttemptsTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("accountId"),
		RangeKey:    pulumi.String("attemptedAt"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("accountId"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("attemptedAt"), Type: pulumi.String("S")},
		},
		Ttl: &dynamodb.TableTtlArgs{
			AttributeName: pulumi.String("expiresAt"),
			Enabled:       pulumi.Bool(true),
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Magic tokens: shared table for every one-shot link we email out
	// (guardian consent A0.4, DSR access/restore A1.1, email-change
	// confirmation A1.1). Rows discriminate via the `kind` attribute. PK=id
	// (opaque random string). TTL on `expiresAt` (epoch seconds) auto-purges
	// abandoned/expired tokens. No PITR; expiring tokens are not data worth
	// recovering.
	magicTokens, err := dynamodb.NewTable(ctx, "magic-tokens-table", &dynamodb.TableArgs{
		Name:        pulumi.String(MagicTokensTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
		},
		Ttl: &dynamodb.TableTtlArgs{
			AttributeName: pulumi.String("expiresAt"),
			Enabled:       pulumi.Bool(true),
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Audit log (GDPR A1.1.4 + A1.2). PK=accountId (S), SK=at (S, RFC3339Nano
	// so rows from the same logical second still sort deterministically by
	// real-world arrival). One row per recorded action; the same stream
	// powers the user's /my-data "Your activity" view and any future
	// admin-side audit dashboard. SSE on. PITR off (retention is bounded
	// to ~24 months by a future sweep, not the recoverable-data category).
	audit, err := dynamodb.NewTable(ctx, "audit-table", &dynamodb.TableArgs{
		Name:        pulumi.String(AuditTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("accountId"),
		RangeKey:    pulumi.String("at"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("accountId"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("at"), Type: pulumi.String("S")},
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Rate-limit table (GDPR A0.7). PK=bucket (S, e.g. "register#<ip>"),
	// SK=at (S, RFC3339Nano). TTL on `expiresAt` (epoch seconds) — rows
	// auto-evict after the rate-limit window so the count stays bounded.
	// IPs are processed under legitimate-interest for fraud/spam prevention
	// (Art. 6(1)(f)); retention ≤ a few minutes makes this proportionate.
	// SSE on; no PITR (short-lived telemetry, not data worth recovering).
	rateLimit, err := dynamodb.NewTable(ctx, "rate-limit-table", &dynamodb.TableArgs{
		Name:        pulumi.String(RateLimitTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("bucket"),
		RangeKey:    pulumi.String("at"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("bucket"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("at"), Type: pulumi.String("S")},
		},
		Ttl: &dynamodb.TableTtlArgs{
			AttributeName: pulumi.String("expiresAt"),
			Enabled:       pulumi.Bool(true),
		},
		ServerSideEncryption: sseEnabled(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	return &Tables{
		Runners:       runners,
		Registrations: registrations,
		Events:        events,
		Races:         races,
		Accounts:      accounts,
		AuthAttempts:  authAttempts,
		MagicTokens:   magicTokens,
		Audit:         audit,
		RateLimit:     rateLimit,
	}, nil
}
