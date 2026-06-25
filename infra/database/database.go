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
)

type Tables struct {
	Runners       *dynamodb.Table
	Registrations *dynamodb.Table
	Events        *dynamodb.Table
	Races         *dynamodb.Table
}

func Setup(ctx *pulumi.Context, provider *aws.Provider) (*Tables, error) {
	var opts []pulumi.ResourceOption
	if provider != nil {
		opts = append(opts, pulumi.Provider(provider))
	}

	runners, err := dynamodb.NewTable(ctx, "runners-table", &dynamodb.TableArgs{
		Name:        pulumi.String(RunnersTableName),
		BillingMode: pulumi.String("PAY_PER_REQUEST"),
		HashKey:     pulumi.String("id"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{Name: pulumi.String("id"), Type: pulumi.String("S")},
			&dynamodb.TableAttributeArgs{Name: pulumi.String("nameDobKey"), Type: pulumi.String("S")},
		},
		GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
			&dynamodb.TableGlobalSecondaryIndexArgs{
				Name:           pulumi.String("byNameDOB"),
				HashKey:        pulumi.String("nameDobKey"),
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

	return &Tables{Runners: runners, Registrations: registrations, Events: events, Races: races}, nil
}
