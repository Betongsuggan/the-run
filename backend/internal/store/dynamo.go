package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

const byNameDOBIndex = "byNameDOB"

type DynamoStore struct {
	client             *dynamodb.Client
	runnersTable       string
	registrationsTable string
}

// NewDynamoStoreFromEnv reads RUNNERS_TABLE_NAME and REGISTRATIONS_TABLE_NAME
// from the environment, loads the AWS SDK's default config, and (if
// AWS_ENDPOINT_URL is set) points the DynamoDB client at it — that's how we
// target LocalStack in local dev.
func NewDynamoStoreFromEnv(ctx context.Context) (*DynamoStore, error) {
	runnersTable := os.Getenv("RUNNERS_TABLE_NAME")
	registrationsTable := os.Getenv("REGISTRATIONS_TABLE_NAME")
	if runnersTable == "" || registrationsTable == "" {
		return nil, errors.New("RUNNERS_TABLE_NAME and REGISTRATIONS_TABLE_NAME must be set")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var optFns []func(*dynamodb.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		optFns = append(optFns, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &DynamoStore{
		client:             dynamodb.NewFromConfig(cfg, optFns...),
		runnersTable:       runnersTable,
		registrationsTable: registrationsTable,
	}, nil
}

type runnerItem struct {
	ID         string `dynamodbav:"id"`
	NameDobKey string `dynamodbav:"nameDobKey"`
	Name       string `dynamodbav:"name"`
	BirthDate  string `dynamodbav:"birthDate"`
	Gender     string `dynamodbav:"gender"`
	CreatedAt  string `dynamodbav:"createdAt"`
}

func (s *DynamoStore) RunnerByNameDOB(ctx context.Context, nameDobKey string) (*models.Runner, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.runnersTable),
		IndexName:              aws.String(byNameDOBIndex),
		KeyConditionExpression: aws.String("nameDobKey = :k"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":k": &dynamodbtypes.AttributeValueMemberS{Value: nameDobKey},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query runners byNameDOB: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, nil
	}

	var item runnerItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return nil, fmt.Errorf("unmarshal runner: %w", err)
	}
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	return &models.Runner{
		ID:        item.ID,
		Name:      item.Name,
		BirthDate: item.BirthDate,
		Gender:    item.Gender,
		CreatedAt: createdAt,
	}, nil
}

func (s *DynamoStore) CreateRunner(ctx context.Context, r models.Runner) error {
	item, err := attributevalue.MarshalMap(runnerItem{
		ID:         r.ID,
		NameDobKey: r.NameDobKey(),
		Name:       r.Name,
		BirthDate:  r.BirthDate,
		Gender:     r.Gender,
		CreatedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal runner: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.runnersTable),
		Item:      item,
	}); err != nil {
		return fmt.Errorf("put runner: %w", err)
	}
	return nil
}

type registrationItem struct {
	RaceID    string `dynamodbav:"raceId"`
	RunnerID  string `dynamodbav:"runnerId"`
	ID        string `dynamodbav:"id"`
	Status    string `dynamodbav:"status"`
	CreatedAt string `dynamodbav:"createdAt"`
}

func (s *DynamoStore) CreateRegistration(ctx context.Context, reg models.Registration) error {
	item, err := attributevalue.MarshalMap(registrationItem{
		RaceID:    reg.RaceID,
		RunnerID:  reg.RunnerID,
		ID:        reg.ID,
		Status:    reg.Status,
		CreatedAt: reg.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.registrationsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(raceId) AND attribute_not_exists(runnerId)"),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ConditionalCheckFailedException" {
			return ErrAlreadyRegistered
		}
		return fmt.Errorf("put registration: %w", err)
	}
	return nil
}
