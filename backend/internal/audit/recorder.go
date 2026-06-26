// Package audit is the per-account activity log (GDPR A1.1.4 + A1.2). One
// Recorder.Record call per recorded action; rows surface back to the user
// via /dsr/me's recentAudit field and (later) to admins via a dedicated
// audit dashboard.
//
// The Recorder interface mirrors email.Sender's shape: production wires
// DynamoRecorder, unit tests and local dev use NoOpRecorder.
package audit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// Recorder writes one row per recorded action and reads them back to power
// the user-facing "Your activity" view. Implementations are best-effort
// on the write side: a failed write logs but doesn't return — the calling
// business logic should never fail because the audit log is unreachable.
// Reads and purges return an error on failure.
type Recorder interface {
	Record(ctx context.Context, row models.AuditRow)
	ListByAccount(ctx context.Context, accountID string, limit int32) ([]models.AuditRow, error)
	// PurgeBefore deletes audit rows whose At is strictly less than the
	// given cutoff. Used by the retention sweep (GDPR A1.3) to roll
	// audit history off after 24 months. Returns the count purged so
	// the sweep can log it; non-fatal on partial failure.
	PurgeBefore(ctx context.Context, before time.Time) (int, error)
}

// NewRecorderFromEnv returns a DynamoRecorder when AUDIT_TABLE_NAME is set,
// otherwise NoOpRecorder. Mirrors email.NewSenderFromEnv's pattern so the
// caller can log which one is active at startup.
func NewRecorderFromEnv(ctx context.Context) (Recorder, bool, error) {
	table := os.Getenv("AUDIT_TABLE_NAME")
	if table == "" {
		return NoOpRecorder{}, false, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("load aws config: %w", err)
	}
	var optFns []func(*dynamodb.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		optFns = append(optFns, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	return &DynamoRecorder{
		client: dynamodb.NewFromConfig(cfg, optFns...),
		table:  table,
	}, true, nil
}

// NoOpRecorder logs the row to stdout and drops it. Suitable for tests and
// local-dev environments where AUDIT_TABLE_NAME isn't set.
type NoOpRecorder struct{}

func (NoOpRecorder) Record(_ context.Context, row models.AuditRow) {
	log.Printf("audit: NoOp record account=%s action=%s actor=%s target=%s/%s",
		row.AccountID, row.Action, row.Actor, row.TargetType, row.TargetID)
}

// ListByAccount returns empty in NoOp mode — the dev environment doesn't
// have an audit table, so the /my-data activity view stays empty too.
// Local devs who want to populate it can point AUDIT_TABLE_NAME at a
// LocalStack table.
func (NoOpRecorder) ListByAccount(_ context.Context, _ string, _ int32) ([]models.AuditRow, error) {
	return nil, nil
}

// PurgeBefore is a no-op in NoOp mode (no table to purge from).
func (NoOpRecorder) PurgeBefore(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

// DynamoRecorder writes rows to the the-run-audit table.
type DynamoRecorder struct {
	client *dynamodb.Client
	table  string
}

type auditItem struct {
	AccountID  string `dynamodbav:"accountId"`
	At         string `dynamodbav:"at"`
	Action     string `dynamodbav:"action"`
	Actor      string `dynamodbav:"actor"`
	TargetType string `dynamodbav:"targetType,omitempty"`
	TargetID   string `dynamodbav:"targetId,omitempty"`
	Summary    string `dynamodbav:"summary,omitempty"`
	Diff       string `dynamodbav:"diff,omitempty"`
}

func (r *DynamoRecorder) Record(ctx context.Context, row models.AuditRow) {
	if row.AccountID == "" {
		// Defensive — every audit row should be account-scoped. A row
		// with an empty PK would Put successfully and pollute account ""
		// queries. Drop it loudly.
		log.Printf("audit: dropping row with empty accountId (action=%s)", row.Action)
		return
	}
	at := row.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	it := auditItem{
		AccountID:  row.AccountID,
		At:         at.UTC().Format(time.RFC3339Nano),
		Action:     row.Action,
		Actor:      row.Actor,
		TargetType: row.TargetType,
		TargetID:   row.TargetID,
		Summary:    row.Summary,
		Diff:       row.Diff,
	}
	item, err := attributevalue.MarshalMap(it)
	if err != nil {
		log.Printf("audit: marshal failed: account=%s action=%s err=%v", row.AccountID, row.Action, err)
		return
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		log.Printf("audit: put failed: account=%s action=%s err=%v", row.AccountID, row.Action, err)
	}
}

// PurgeBefore scans the audit table for rows older than `before` and
// deletes them. Pagination follows DynamoDB's LastEvaluatedKey contract
// since Scan caps at 1 MB per page; the audit table can grow well past
// that even at our scale.
//
// We Scan rather than Query because the table's PK is accountId — there
// is no global secondary index on `at` to query against. A daily sweep
// over an audit table at our scale (estimate: low-MB per year) is cheap
// in DynamoDB on-demand pricing terms, so we don't optimise further.
func (r *DynamoRecorder) PurgeBefore(ctx context.Context, before time.Time) (int, error) {
	cutoff := before.UTC().Format(time.RFC3339Nano)
	deleted := 0
	var lastKey map[string]dynamodbtypes.AttributeValue
	for {
		out, err := r.client.Scan(ctx, &dynamodb.ScanInput{
			TableName:        aws.String(r.table),
			FilterExpression: aws.String("#at < :cutoff"),
			ExpressionAttributeNames: map[string]string{
				"#at": "at",
			},
			ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
				":cutoff": &dynamodbtypes.AttributeValueMemberS{Value: cutoff},
			},
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return deleted, fmt.Errorf("scan audit: %w", err)
		}
		for _, item := range out.Items {
			var it auditItem
			if err := attributevalue.UnmarshalMap(item, &it); err != nil {
				continue
			}
			if it.AccountID == "" || it.At == "" {
				continue
			}
			_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(r.table),
				Key: map[string]dynamodbtypes.AttributeValue{
					"accountId": &dynamodbtypes.AttributeValueMemberS{Value: it.AccountID},
					"at":        &dynamodbtypes.AttributeValueMemberS{Value: it.At},
				},
			})
			if err != nil {
				log.Printf("audit: purge delete failed: account=%s at=%s err=%v", it.AccountID, it.At, err)
				continue
			}
			deleted++
		}
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}
	return deleted, nil
}

// ListByAccount returns the most recent `limit` audit rows for an account,
// newest first. Limit is enforced by Query's Limit parameter; the caller
// chooses the page size.
func (r *DynamoRecorder) ListByAccount(ctx context.Context, accountID string, limit int32) ([]models.AuditRow, error) {
	if accountID == "" {
		return nil, errors.New("accountId required")
	}
	if limit <= 0 {
		limit = 50
	}
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("accountId = :a"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":a": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
		},
		ScanIndexForward: aws.Bool(false), // newest-first
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	rows := make([]models.AuditRow, 0, len(out.Items))
	for _, raw := range out.Items {
		var it auditItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, it.At)
		rows = append(rows, models.AuditRow{
			AccountID:  it.AccountID,
			At:         at,
			Action:     it.Action,
			Actor:      it.Actor,
			TargetType: it.TargetType,
			TargetID:   it.TargetID,
			Summary:    it.Summary,
			Diff:       it.Diff,
		})
	}
	return rows, nil
}
