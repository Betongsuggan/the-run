package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// RecordAuthAttempt writes one failed-login row keyed by (accountID, RFC3339
// timestamp). DynamoDB's TTL daemon cleans them up automatically; CountActive
// also filters by expiresAt to handle the eventual-consistency lag.
func (s *DynamoStore) RecordAuthAttempt(ctx context.Context, accountID string, at time.Time, ttl time.Duration) error {
	expires := at.Add(ttl).Unix()
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.authAttemptsTable),
		Item: map[string]dynamodbtypes.AttributeValue{
			"accountId":   &dynamodbtypes.AttributeValueMemberS{Value: accountID},
			"attemptedAt": &dynamodbtypes.AttributeValueMemberS{Value: at.UTC().Format(time.RFC3339Nano)},
			"expiresAt":   &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", expires)},
		},
	})
	if err != nil {
		return fmt.Errorf("record auth attempt: %w", err)
	}
	return nil
}

func (s *DynamoStore) CountActiveAuthAttempts(ctx context.Context, accountID string, now time.Time) (int, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.authAttemptsTable),
		KeyConditionExpression: aws.String("accountId = :a"),
		FilterExpression:       aws.String("expiresAt > :n"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":a": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
			":n": &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("count auth attempts: %w", err)
	}
	return int(out.Count), nil
}

func (s *DynamoStore) ClearAuthAttempts(ctx context.Context, accountID string) error {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.authAttemptsTable),
		KeyConditionExpression: aws.String("accountId = :a"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":a": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
		},
		ProjectionExpression: aws.String("accountId, attemptedAt"),
	})
	if err != nil {
		return fmt.Errorf("query auth attempts: %w", err)
	}
	if len(out.Items) == 0 {
		return nil
	}
	// BatchWriteItem caps at 25 deletes per request; chunk if needed.
	for start := 0; start < len(out.Items); start += 25 {
		end := min(start+25, len(out.Items))
		writes := make([]dynamodbtypes.WriteRequest, 0, end-start)
		for _, item := range out.Items[start:end] {
			writes = append(writes, dynamodbtypes.WriteRequest{
				DeleteRequest: &dynamodbtypes.DeleteRequest{
					Key: map[string]dynamodbtypes.AttributeValue{
						"accountId":   item["accountId"],
						"attemptedAt": item["attemptedAt"],
					},
				},
			})
		}
		if _, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]dynamodbtypes.WriteRequest{
				s.authAttemptsTable: writes,
			},
		}); err != nil {
			return fmt.Errorf("delete auth attempts: %w", err)
		}
	}
	return nil
}
