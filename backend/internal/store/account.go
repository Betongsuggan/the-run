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
	ID                   string       `dynamodbav:"id"`
	Email                string       `dynamodbav:"email"`
	PasswordHash         string       `dynamodbav:"passwordHash,omitempty"`
	MFASecret            string       `dynamodbav:"mfaSecret,omitempty"`
	IsAdmin              bool         `dynamodbav:"isAdmin"`
	Marketing            *consentItem `dynamodbav:"marketing,omitempty"`
	Locale               string       `dynamodbav:"locale,omitempty"`
	CreatedAt            string       `dynamodbav:"createdAt"`
	LastLoginAt          string       `dynamodbav:"lastLoginAt,omitempty"`
	DeletionPendingUntil string       `dynamodbav:"deletionPendingUntil,omitempty"`
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
	if item.DeletionPendingUntil != "" {
		t, err := time.Parse(time.RFC3339, item.DeletionPendingUntil)
		if err == nil {
			a.DeletionPendingUntil = &t
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
	if a.DeletionPendingUntil != nil {
		item.DeletionPendingUntil = a.DeletionPendingUntil.UTC().Format(time.RFC3339)
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
		// Data-integrity error, not user-facing. Don't embed the email so this
		// can't accidentally leak into logs (GDPR A0.5).
		return nil, errors.New("email sentinel row is missing accountId")
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

// ListAccounts returns every primary account row (excludes EMAIL# sentinel
// rows by filtering on the `kind` attribute). Used by the retention Lambda
// to find accounts past their grace window. Order is unspecified.
func (s *DynamoStore) ListAccounts(ctx context.Context) ([]models.Account, error) {
	items, err := s.scanAll(ctx, s.accountsTable)
	if err != nil {
		return nil, fmt.Errorf("scan accounts: %w", err)
	}
	accounts := make([]models.Account, 0, len(items))
	for _, raw := range items {
		// Skip sentinel rows — they have id="EMAIL#..." and kind="email-sentinel".
		var it accountItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			continue
		}
		if it.Kind != "account" {
			continue
		}
		accounts = append(accounts, accountFromItem(it))
	}
	return accounts, nil
}

// DeleteAccount removes the primary account row AND its email sentinel in
// one TransactWriteItems call. Used by the retention Lambda when a soft-
// deleted account has aged past its grace window. The retention job
// nulls-out PII first (via UpdateAccount with empty fields), then calls
// DeleteAccount to drop the row entirely — we'd lose the email reference
// otherwise.
func (s *DynamoStore) DeleteAccount(ctx context.Context, accountID, email string) error {
	if accountID == "" {
		return errors.New("accountId required")
	}
	transactItems := []dynamodbtypes.TransactWriteItem{
		{
			Delete: &dynamodbtypes.Delete{
				TableName: aws.String(s.accountsTable),
				Key: map[string]dynamodbtypes.AttributeValue{
					"id": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
				},
			},
		},
	}
	if email != "" {
		transactItems = append(transactItems, dynamodbtypes.TransactWriteItem{
			Delete: &dynamodbtypes.Delete{
				TableName: aws.String(s.accountsTable),
				Key: map[string]dynamodbtypes.AttributeValue{
					"id": &dynamodbtypes.AttributeValueMemberS{Value: emailSentinelID(email)},
				},
			},
		})
	}
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

// ChangeAccountEmail atomically rewrites the email sentinel and updates the
// account's email field. The three-op TransactWriteItems guarantees no
// partial state — either the new sentinel takes hold and the account row is
// updated, or nothing happens.
//
// Returns ErrAlreadyExists if the new email is already on another account
// (the put-with-attribute_not_exists fails). Returns ErrNotFound if the
// account row went away between the read and this call (unusual; account
// row is deleted only by the retention job).
func (s *DynamoStore) ChangeAccountEmail(ctx context.Context, accountID, oldEmail, newEmail string) error {
	if accountID == "" || oldEmail == "" || newEmail == "" {
		return errors.New("accountId, oldEmail, newEmail are required")
	}
	oldNorm := models.NormalizeEmail(oldEmail)
	newNorm := models.NormalizeEmail(newEmail)
	if oldNorm == newNorm {
		return nil
	}

	// New sentinel row pointing at this account.
	newSentinel, err := attributevalue.MarshalMap(emailSentinelItem{
		ID:        emailSentinelID(newNorm),
		AccountID: accountID,
		Kind:      "email-sentinel",
	})
	if err != nil {
		return fmt.Errorf("marshal new sentinel: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbtypes.TransactWriteItem{
			// (1) Delete the OLD sentinel — conditional on it still pointing
			// to this account (defence against a concurrent change).
			{
				Delete: &dynamodbtypes.Delete{
					TableName: aws.String(s.accountsTable),
					Key: map[string]dynamodbtypes.AttributeValue{
						"id": &dynamodbtypes.AttributeValueMemberS{Value: emailSentinelID(oldNorm)},
					},
					ConditionExpression: aws.String("accountId = :a"),
					ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
						":a": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
					},
				},
			},
			// (2) Put the NEW sentinel — must not already exist (would
			// collide with another account or with this account's pre-
			// existing claim).
			{
				Put: &dynamodbtypes.Put{
					TableName:           aws.String(s.accountsTable),
					Item:                newSentinel,
					ConditionExpression: aws.String("attribute_not_exists(id)"),
				},
			},
			// (3) Update the account row's email field. Conditional on the
			// row still existing AND still showing the old email — guards
			// against weird concurrent edits.
			{
				Update: &dynamodbtypes.Update{
					TableName: aws.String(s.accountsTable),
					Key: map[string]dynamodbtypes.AttributeValue{
						"id": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
					},
					UpdateExpression:    aws.String("SET email = :new"),
					ConditionExpression: aws.String("attribute_exists(id) AND email = :old"),
					ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
						":new": &dynamodbtypes.AttributeValueMemberS{Value: newNorm},
						":old": &dynamodbtypes.AttributeValueMemberS{Value: oldNorm},
					},
				},
			},
		},
	})
	if err != nil {
		if isTransactionCanceled(err) {
			// Most likely cause: the new sentinel collided. Race with another
			// account claiming the same address. Surface as ErrAlreadyExists
			// so the handler maps to 409.
			return ErrAlreadyExists
		}
		return fmt.Errorf("swap account email: %w", err)
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
