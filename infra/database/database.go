// Package database provisions the DynamoDB tables backing the API. The same
// Setup is invoked by both the production AWS stack and the LocalStack-targeted
// local stack — `provider` is the difference (pass nil for the default
// provider).
package database

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	RunnersTableName       = "the-run-runners"
	RegistrationsTableName = "the-run-registrations"
	EventsTableName        = "the-run-events"
	RacesTableName         = "the-run-races"
	AccountsTableName      = "the-run-accounts"
	AuthAttemptsTableName  = "the-run-auth-attempts"
)

type Tables struct {
	Runners       *dynamodb.Table
	Registrations *dynamodb.Table
	Events        *dynamodb.Table
	Races         *dynamodb.Table
	Accounts      *dynamodb.Table
	AuthAttempts  *dynamodb.Table
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
		PointInTimeRecovery: &dynamodb.TablePointInTimeRecoveryArgs{
			Enabled: pulumi.Bool(true),
		},
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
	}, nil
}
