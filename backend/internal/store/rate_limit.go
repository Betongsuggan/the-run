package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// RecordRateLimitHit writes one row keyed by (bucket, RFC3339Nano timestamp).
// `bucket` is opaque to the store — callers pick the namespacing
// convention (e.g. "register#<ip>"). The TTL is expressed in seconds
// since epoch on `expiresAt`; DynamoDB's TTL daemon cleans up old rows
// lazily, and CountActiveRateLimitHits also filters by it to handle the
// (typically minutes-long) eventual-consistency lag.
func (s *DynamoStore) RecordRateLimitHit(ctx context.Context, bucket string, at time.Time, ttl time.Duration) error {
	expires := at.Add(ttl).Unix()
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.rateLimitTable),
		Item: map[string]dynamodbtypes.AttributeValue{
			"bucket":    &dynamodbtypes.AttributeValueMemberS{Value: bucket},
			"at":        &dynamodbtypes.AttributeValueMemberS{Value: at.UTC().Format(time.RFC3339Nano)},
			"expiresAt": &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", expires)},
		},
	})
	if err != nil {
		return fmt.Errorf("record rate limit hit: %w", err)
	}
	return nil
}

// CountActiveRateLimitHits returns the number of non-expired rows for the
// bucket. The FilterExpression bridges the TTL-daemon lag — DynamoDB
// promises TTL deletion within ~48h, so we can't trust the row count alone.
func (s *DynamoStore) CountActiveRateLimitHits(ctx context.Context, bucket string, now time.Time) (int, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.rateLimitTable),
		KeyConditionExpression: aws.String("bucket = :b"),
		FilterExpression:       aws.String("expiresAt > :n"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":b": &dynamodbtypes.AttributeValueMemberS{Value: bucket},
			":n": &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("count rate limit hits: %w", err)
	}
	return int(out.Count), nil
}
