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

// guardianTokenItem is the on-the-wire row. ExpiresAt is duplicated as an
// RFC3339 string (for humans / debugging) and as an epoch-seconds number
// (DynamoDB TTL requires a Number attribute named exactly as configured).
type guardianTokenItem struct {
	ID                string `dynamodbav:"id"`
	RegistrationID    string `dynamodbav:"registrationId"`
	GuardianAccountID string `dynamodbav:"guardianAccountId"`
	ExpiresAt         string `dynamodbav:"expiresAtIso"`
	ExpiresAtEpoch    int64  `dynamodbav:"expiresAt"`
	UsedAt            string `dynamodbav:"usedAt,omitempty"`
	CreatedAt         string `dynamodbav:"createdAt"`
}

func guardianTokenFromItem(it guardianTokenItem) models.GuardianToken {
	created, _ := time.Parse(time.RFC3339, it.CreatedAt)
	expires, _ := time.Parse(time.RFC3339, it.ExpiresAt)
	t := models.GuardianToken{
		ID:                it.ID,
		RegistrationID:    it.RegistrationID,
		GuardianAccountID: it.GuardianAccountID,
		ExpiresAt:         expires,
		CreatedAt:         created,
	}
	if it.UsedAt != "" {
		if u, err := time.Parse(time.RFC3339, it.UsedAt); err == nil {
			t.UsedAt = &u
		}
	}
	return t
}

// CreateGuardianToken writes a fresh consent token. ID is the random opaque
// string the caller generated (typically 32+ bytes of entropy, base64url).
func (s *DynamoStore) CreateGuardianToken(ctx context.Context, t models.GuardianToken) error {
	it := guardianTokenItem{
		ID:                t.ID,
		RegistrationID:    t.RegistrationID,
		GuardianAccountID: t.GuardianAccountID,
		ExpiresAt:         t.ExpiresAt.UTC().Format(time.RFC3339),
		ExpiresAtEpoch:    t.ExpiresAt.Unix(),
		CreatedAt:         t.CreatedAt.UTC().Format(time.RFC3339),
	}
	item, err := attributevalue.MarshalMap(it)
	if err != nil {
		return fmt.Errorf("marshal guardian token: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.guardianTokensTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("put guardian token: %w", err)
	}
	return nil
}

// GetGuardianToken returns ErrNotFound when the token doesn't exist (revoked,
// expired-and-purged, or never issued). Expired-but-still-present tokens are
// returned as-is — the caller checks ExpiresAt against time.Now().
func (s *DynamoStore) GetGuardianToken(ctx context.Context, id string) (*models.GuardianToken, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.guardianTokensTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get guardian token: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var it guardianTokenItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return nil, fmt.Errorf("unmarshal guardian token: %w", err)
	}
	t := guardianTokenFromItem(it)
	return &t, nil
}

// MarkGuardianTokenUsed stamps the usedAt timestamp. The conditional check
// blocks double-use races: if another request marked it used in the meantime,
// the second caller gets ErrAlreadyExists (which the handler maps to a clear
// "token already used" error).
func (s *DynamoStore) MarkGuardianTokenUsed(ctx context.Context, id string, at time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.guardianTokensTable),
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
		return fmt.Errorf("mark guardian token used: %w", err)
	}
	return nil
}

// DeleteGuardianToken removes a token outright — used when an admin revokes
// before the guardian clicks.
func (s *DynamoStore) DeleteGuardianToken(ctx context.Context, id string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.guardianTokensTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete guardian token: %w", err)
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
