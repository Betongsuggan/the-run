package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// emailSentinelPrefix is the PK prefix for the per-email sentinel row that
// guards uniqueness. Primary account rows use UUIDs (no prefix), so the two
// namespaces never collide.
const emailSentinelPrefix = "EMAIL#"

func emailSentinelID(email string) string {
	return emailSentinelPrefix + models.NormalizeEmail(email)
}

type accountItem struct {
	ID           string       `dynamodbav:"id"`
	Email        string       `dynamodbav:"email"`
	PasswordHash string       `dynamodbav:"passwordHash,omitempty"`
	MFASecret    string       `dynamodbav:"mfaSecret,omitempty"`
	IsAdmin      bool         `dynamodbav:"isAdmin"`
	Marketing    *consentItem `dynamodbav:"marketing,omitempty"`
	Locale       string       `dynamodbav:"locale,omitempty"`
	CreatedAt    string       `dynamodbav:"createdAt"`
	LastLoginAt  string       `dynamodbav:"lastLoginAt,omitempty"`
	// Discriminator so a Scan can tell primary rows from sentinel rows.
	Kind string `dynamodbav:"kind"`
}

type consentItem struct {
	Granted       bool   `dynamodbav:"granted"`
	At            string `dynamodbav:"at"`
	PolicyVersion string `dynamodbav:"policyVersion"`
}

type emailSentinelItem struct {
	ID        string `dynamodbav:"id"`
	AccountID string `dynamodbav:"accountId"`
	Kind      string `dynamodbav:"kind"`
}

func accountFromItem(item accountItem) models.Account {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	a := models.Account{
		ID:           item.ID,
		Email:        item.Email,
		PasswordHash: item.PasswordHash,
		MFASecret:    item.MFASecret,
		IsAdmin:      item.IsAdmin,
		Locale:       item.Locale,
		CreatedAt:    createdAt,
	}
	if item.Marketing != nil {
		at, _ := time.Parse(time.RFC3339, item.Marketing.At)
		a.Consents.Marketing = models.Consent{
			Granted:       item.Marketing.Granted,
			At:            at,
			PolicyVersion: item.Marketing.PolicyVersion,
		}
	}
	if item.LastLoginAt != "" {
		t, err := time.Parse(time.RFC3339, item.LastLoginAt)
		if err == nil {
			a.LastLoginAt = &t
		}
	}
	return a
}

func itemFromAccount(a models.Account) accountItem {
	item := accountItem{
		ID:           a.ID,
		Email:        models.NormalizeEmail(a.Email),
		PasswordHash: a.PasswordHash,
		MFASecret:    a.MFASecret,
		IsAdmin:      a.IsAdmin,
		Locale:       a.Locale,
		CreatedAt:    a.CreatedAt.UTC().Format(time.RFC3339),
		Kind:         "account",
	}
	if !a.Consents.Marketing.At.IsZero() || a.Consents.Marketing.PolicyVersion != "" {
		item.Marketing = &consentItem{
			Granted:       a.Consents.Marketing.Granted,
			At:            a.Consents.Marketing.At.UTC().Format(time.RFC3339),
			PolicyVersion: a.Consents.Marketing.PolicyVersion,
		}
	}
	if a.LastLoginAt != nil {
		item.LastLoginAt = a.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return item
}

func (s *DynamoStore) GetAccountByID(ctx context.Context, id string) (*models.Account, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.accountsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item accountItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal account: %w", err)
	}
	a := accountFromItem(item)
	return &a, nil
}

func (s *DynamoStore) GetAccountByEmail(ctx context.Context, email string) (*models.Account, error) {
	sentOut, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.accountsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: emailSentinelID(email)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get email sentinel: %w", err)
	}
	if sentOut.Item == nil {
		// Not registered yet — caller's signal for find-or-create.
		return nil, nil
	}
	var sentinel emailSentinelItem
	if err := attributevalue.UnmarshalMap(sentOut.Item, &sentinel); err != nil {
		return nil, fmt.Errorf("unmarshal sentinel: %w", err)
	}
	if sentinel.AccountID == "" {
		return nil, fmt.Errorf("sentinel for %q missing accountId", email)
	}
	return s.GetAccountByID(ctx, sentinel.AccountID)
}

func (s *DynamoStore) CreateAccount(ctx context.Context, a models.Account) error {
	if a.ID == "" {
		return errors.New("account id required")
	}
	if a.Email == "" {
		return errors.New("account email required")
	}

	primary, err := attributevalue.MarshalMap(itemFromAccount(a))
	if err != nil {
		return fmt.Errorf("marshal account: %w", err)
	}
	sentinel, err := attributevalue.MarshalMap(emailSentinelItem{
		ID:        emailSentinelID(a.Email),
		AccountID: a.ID,
		Kind:      "email-sentinel",
	})
	if err != nil {
		return fmt.Errorf("marshal sentinel: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbtypes.TransactWriteItem{
			{
				Put: &dynamodbtypes.Put{
					TableName:           aws.String(s.accountsTable),
					Item:                primary,
					ConditionExpression: aws.String("attribute_not_exists(id)"),
				},
			},
			{
				Put: &dynamodbtypes.Put{
					TableName:           aws.String(s.accountsTable),
					Item:                sentinel,
					ConditionExpression: aws.String("attribute_not_exists(id)"),
				},
			},
		},
	})
	if err != nil {
		if isTransactionCanceled(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (s *DynamoStore) UpdateAccount(ctx context.Context, a models.Account) error {
	item, err := attributevalue.MarshalMap(itemFromAccount(a))
	if err != nil {
		return fmt.Errorf("marshal account: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.accountsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}

func (s *DynamoStore) TouchAccountLastLogin(ctx context.Context, id string, at time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.accountsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET lastLoginAt = :t"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":t": &dynamodbtypes.AttributeValueMemberS{Value: at.UTC().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("touch lastLoginAt: %w", err)
	}
	return nil
}

// isTransactionCanceled distinguishes a conditional-check failure inside a
// TransactWriteItems from other transport errors. AWS reports this as
// TransactionCanceledException with cancellation reasons per item.
func isTransactionCanceled(err error) bool {
	var canceled *dynamodbtypes.TransactionCanceledException
	return errors.As(err, &canceled)
}
