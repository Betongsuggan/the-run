package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// adminInvitationItem mirrors the magic-token table layout: the row carries
// both the RFC3339 expiresAtIso (for humans) and an epoch-seconds expiresAt
// (consumed by the DynamoDB TTL).
type adminInvitationItem struct {
	TokenHash      string `dynamodbav:"tokenHash"`
	Email          string `dynamodbav:"email"`
	Locale         string `dynamodbav:"locale,omitempty"`
	InvitedByID    string `dynamodbav:"invitedById,omitempty"`
	InvitedByMail  string `dynamodbav:"invitedByMail,omitempty"`
	ExpiresAt      string `dynamodbav:"expiresAtIso"`
	ExpiresAtEpoch int64  `dynamodbav:"expiresAt"`
	UsedAt         string `dynamodbav:"usedAt,omitempty"`
	CreatedAt      string `dynamodbav:"createdAt"`
}

func adminInvitationFromItem(it adminInvitationItem) models.AdminInvitation {
	created, _ := time.Parse(time.RFC3339, it.CreatedAt)
	expires, _ := time.Parse(time.RFC3339, it.ExpiresAt)
	inv := models.AdminInvitation{
		TokenHash:     it.TokenHash,
		Email:         it.Email,
		Locale:        it.Locale,
		InvitedByID:   it.InvitedByID,
		InvitedByMail: it.InvitedByMail,
		ExpiresAt:     expires,
		CreatedAt:     created,
	}
	if it.UsedAt != "" {
		if u, err := time.Parse(time.RFC3339, it.UsedAt); err == nil {
			inv.UsedAt = &u
		}
	}
	return inv
}

func (s *DynamoStore) CreateAdminInvitation(ctx context.Context, inv models.AdminInvitation) error {
	it := adminInvitationItem{
		TokenHash:      inv.TokenHash,
		Email:          models.NormalizeEmail(inv.Email),
		Locale:         inv.Locale,
		InvitedByID:    inv.InvitedByID,
		InvitedByMail:  inv.InvitedByMail,
		ExpiresAt:      inv.ExpiresAt.UTC().Format(time.RFC3339),
		ExpiresAtEpoch: inv.ExpiresAt.Unix(),
		CreatedAt:      inv.CreatedAt.UTC().Format(time.RFC3339),
	}
	item, err := attributevalue.MarshalMap(it)
	if err != nil {
		return fmt.Errorf("marshal admin invitation: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.adminInvitationsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(tokenHash)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("put admin invitation: %w", err)
	}
	return nil
}

func (s *DynamoStore) GetAdminInvitationByTokenHash(ctx context.Context, tokenHash string) (*models.AdminInvitation, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.adminInvitationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"tokenHash": &dynamodbtypes.AttributeValueMemberS{Value: tokenHash},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get admin invitation: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var it adminInvitationItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return nil, fmt.Errorf("unmarshal admin invitation: %w", err)
	}
	inv := adminInvitationFromItem(it)
	return &inv, nil
}

// ListAdminInvitations returns every invitation row, newest first. Used by the
// admin /users page to show pending invitations. Expired but un-purged rows
// are included; the handler renders them with a clear "expired" state.
func (s *DynamoStore) ListAdminInvitations(ctx context.Context) ([]models.AdminInvitation, error) {
	items, err := s.scanAll(ctx, s.adminInvitationsTable)
	if err != nil {
		return nil, fmt.Errorf("scan admin invitations: %w", err)
	}
	invs := make([]models.AdminInvitation, 0, len(items))
	for _, raw := range items {
		var it adminInvitationItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			continue
		}
		invs = append(invs, adminInvitationFromItem(it))
	}
	sort.Slice(invs, func(i, j int) bool {
		return invs[i].CreatedAt.After(invs[j].CreatedAt)
	})
	return invs, nil
}

func (s *DynamoStore) MarkAdminInvitationUsed(ctx context.Context, tokenHash string, at time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.adminInvitationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"tokenHash": &dynamodbtypes.AttributeValueMemberS{Value: tokenHash},
		},
		UpdateExpression:    aws.String("SET usedAt = :u"),
		ConditionExpression: aws.String("attribute_exists(tokenHash) AND attribute_not_exists(usedAt)"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":u": &dynamodbtypes.AttributeValueMemberS{Value: at.UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("mark admin invitation used: %w", err)
	}
	return nil
}

func (s *DynamoStore) DeleteAdminInvitation(ctx context.Context, tokenHash string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.adminInvitationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"tokenHash": &dynamodbtypes.AttributeValueMemberS{Value: tokenHash},
		},
	})
	if err != nil {
		return fmt.Errorf("delete admin invitation: %w", err)
	}
	return nil
}
