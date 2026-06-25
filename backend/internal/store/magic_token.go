package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// magicTokenItem is the on-the-wire row. ExpiresAt is duplicated as an RFC3339
// string (for humans / debugging) and as an epoch-seconds number (DynamoDB TTL
// requires a Number attribute named exactly as configured).
type magicTokenItem struct {
	ID             string `dynamodbav:"id"`
	Kind           string `dynamodbav:"kind"`
	AccountID      string `dynamodbav:"accountId"`
	ContextID      string `dynamodbav:"contextId,omitempty"`
	ExpiresAt      string `dynamodbav:"expiresAtIso"`
	ExpiresAtEpoch int64  `dynamodbav:"expiresAt"`
	UsedAt         string `dynamodbav:"usedAt,omitempty"`
	CreatedAt      string `dynamodbav:"createdAt"`
}

func magicTokenFromItem(it magicTokenItem) models.MagicToken {
	created, _ := time.Parse(time.RFC3339, it.CreatedAt)
	expires, _ := time.Parse(time.RFC3339, it.ExpiresAt)
	t := models.MagicToken{
		Kind:      models.TokenKind(it.Kind),
		ID:        it.ID,
		AccountID: it.AccountID,
		ContextID: it.ContextID,
		ExpiresAt: expires,
		CreatedAt: created,
	}
	if it.UsedAt != "" {
		if u, err := time.Parse(time.RFC3339, it.UsedAt); err == nil {
			t.UsedAt = &u
		}
	}
	return t
}

// CreateMagicToken writes a fresh magic-link token. ID is the random opaque
// string the caller generated (typically 32+ bytes of entropy, base64url).
// Kind discriminates between guardian / dsr_access / dsr_restore /
// email_change — see models.TokenKind for semantics.
func (s *DynamoStore) CreateMagicToken(ctx context.Context, t models.MagicToken) error {
	it := magicTokenItem{
		ID:             t.ID,
		Kind:           string(t.Kind),
		AccountID:      t.AccountID,
		ContextID:      t.ContextID,
		ExpiresAt:      t.ExpiresAt.UTC().Format(time.RFC3339),
		ExpiresAtEpoch: t.ExpiresAt.Unix(),
		CreatedAt:      t.CreatedAt.UTC().Format(time.RFC3339),
	}
	item, err := attributevalue.MarshalMap(it)
	if err != nil {
		return fmt.Errorf("marshal magic token: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.magicTokensTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("put magic token: %w", err)
	}
	return nil
}

// GetMagicToken returns ErrNotFound when the token doesn't exist (revoked,
// expired-and-purged, or never issued). Expired-but-still-present tokens are
// returned as-is — the caller checks ExpiresAt against time.Now() and the Kind
// against the operation being performed.
func (s *DynamoStore) GetMagicToken(ctx context.Context, id string) (*models.MagicToken, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.magicTokensTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get magic token: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var it magicTokenItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return nil, fmt.Errorf("unmarshal magic token: %w", err)
	}
	t := magicTokenFromItem(it)
	return &t, nil
}

// MarkMagicTokenUsed stamps the usedAt timestamp. The conditional check blocks
// double-use races: if another request marked it used in the meantime, the
// second caller gets ErrAlreadyExists (which the handler maps to a clear
// "token already used" error).
func (s *DynamoStore) MarkMagicTokenUsed(ctx context.Context, id string, at time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.magicTokensTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:    aws.String("SET usedAt = :u"),
		ConditionExpression: aws.String("attribute_exists(id) AND attribute_not_exists(usedAt)"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":u": &dynamodbtypes.AttributeValueMemberS{Value: at.UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("mark magic token used: %w", err)
	}
	return nil
}

// DeleteMagicToken removes a token outright — used when an admin revokes
// before the user clicks, or as part of the retention purge for an erased
// account.
func (s *DynamoStore) DeleteMagicToken(ctx context.Context, id string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.magicTokensTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete magic token: %w", err)
	}
	return nil
}

// UpdateRegistrationStatus is a focused helper used by the guardian-consent
// flow to flip a registration from pending_guardian_consent → received. It
// reuses the broader UpdateRegistration only via the Status field, but adds
// the by-ID lookup the existing call doesn't expose.
func (s *DynamoStore) UpdateRegistrationStatus(ctx context.Context, registrationID, status string) error {
	reg, err := s.GetRegistrationByID(ctx, registrationID)
	if err != nil {
		return err
	}
	_, err = s.UpdateRegistration(ctx, RegistrationUpdate{
		RaceID:   reg.RaceID,
		RunnerID: reg.RunnerID,
		Status:   &status,
	})
	return err
}
