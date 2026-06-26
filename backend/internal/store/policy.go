package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// slugSentinelPrefix mirrors emailSentinelPrefix in account.go: a sentinel
// row in the policies table reserving the slug. Primary policy rows use
// UUIDs (no prefix); the two namespaces never collide.
const slugSentinelPrefix = "SLUG#"

func slugSentinelID(slug string) string {
	return slugSentinelPrefix + slug
}

type policyItem struct {
	ID            string `dynamodbav:"id"`
	Slug          string `dynamodbav:"slug"`
	Status        string `dynamodbav:"status"`
	Revision      int    `dynamodbav:"revision"`
	EffectiveFrom string `dynamodbav:"effectiveFrom"`
	BodySv        string `dynamodbav:"bodySv"`
	BodyEn        string `dynamodbav:"bodyEn"`
	CreatedAt     string `dynamodbav:"createdAt"`
	UpdatedAt     string `dynamodbav:"updatedAt"`
	PublishedAt   string `dynamodbav:"publishedAt,omitempty"`
	PublishedBy   string `dynamodbav:"publishedBy,omitempty"`
	LastEditedBy  string `dynamodbav:"lastEditedBy,omitempty"`
	// Discriminator so a Scan can tell primary rows from sentinel rows.
	Kind string `dynamodbav:"kind"`
}

type slugSentinelItem struct {
	ID       string `dynamodbav:"id"`
	PolicyID string `dynamodbav:"policyId"`
	Kind     string `dynamodbav:"kind"`
}

type policyRevisionItem struct {
	PolicyID  string `dynamodbav:"policyId"`
	Revision  string `dynamodbav:"revision"`
	BodySv    string `dynamodbav:"bodySv"`
	BodyEn    string `dynamodbav:"bodyEn"`
	EditedAt  string `dynamodbav:"editedAt"`
	EditedBy  string `dynamodbav:"editedBy,omitempty"`
	Note      string `dynamodbav:"note,omitempty"`
	Published bool   `dynamodbav:"published"`
}

func policyFromItem(item policyItem) models.Policy {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, item.UpdatedAt)
	effectiveFrom, _ := time.Parse(time.RFC3339, item.EffectiveFrom)
	p := models.Policy{
		ID:            item.ID,
		Slug:          item.Slug,
		Status:        models.PolicyStatus(item.Status),
		Revision:      item.Revision,
		EffectiveFrom: effectiveFrom,
		BodySv:        item.BodySv,
		BodyEn:        item.BodyEn,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		PublishedBy:   item.PublishedBy,
		LastEditedBy:  item.LastEditedBy,
	}
	if item.PublishedAt != "" {
		t, err := time.Parse(time.RFC3339, item.PublishedAt)
		if err == nil {
			p.PublishedAt = &t
		}
	}
	return p
}

func itemFromPolicy(p models.Policy) policyItem {
	item := policyItem{
		ID:            p.ID,
		Slug:          p.Slug,
		Status:        string(p.Status),
		Revision:      p.Revision,
		EffectiveFrom: p.EffectiveFrom.UTC().Format(time.RFC3339),
		BodySv:        p.BodySv,
		BodyEn:        p.BodyEn,
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.UTC().Format(time.RFC3339),
		PublishedBy:   p.PublishedBy,
		LastEditedBy:  p.LastEditedBy,
		Kind:          "policy",
	}
	if p.PublishedAt != nil {
		item.PublishedAt = p.PublishedAt.UTC().Format(time.RFC3339)
	}
	return item
}

// revisionSortKey turns the integer revision into a zero-padded 10-digit
// string so the DynamoDB sort key orders revisions numerically when sorted
// lexicographically.
func revisionSortKey(revision int) string {
	return fmt.Sprintf("%010d", revision)
}

func policyRevisionFromItem(item policyRevisionItem) models.PolicyRevision {
	editedAt, _ := time.Parse(time.RFC3339, item.EditedAt)
	var revision int
	fmt.Sscanf(item.Revision, "%d", &revision)
	return models.PolicyRevision{
		PolicyID:  item.PolicyID,
		Revision:  revision,
		BodySv:    item.BodySv,
		BodyEn:    item.BodyEn,
		EditedAt:  editedAt,
		EditedBy:  item.EditedBy,
		Note:      item.Note,
		Published: item.Published,
	}
}

// CreatePolicyDraft writes both the primary row and the slug sentinel in one
// TransactWriteItems, then appends revision 1 to the history table. The two
// writes aren't fully atomic — if the second one fails after the first
// succeeds the policy will look correct but lack a snapshot for revision 1.
// We accept that risk; the history is recoverable from the head body in that
// edge case and the failure surface is operator-level (admin retries the
// draft creation).
func (s *DynamoStore) CreatePolicyDraft(ctx context.Context, p models.Policy) error {
	if p.ID == "" {
		return errors.New("policy id required")
	}
	if p.Slug == "" {
		return errors.New("policy slug required")
	}
	if p.BodySv == "" || p.BodyEn == "" {
		return errors.New("policy bodies required (sv + en)")
	}
	if p.Status == "" {
		p.Status = models.PolicyStatusDraft
	}
	if p.Revision == 0 {
		p.Revision = 1
	}

	primary, err := attributevalue.MarshalMap(itemFromPolicy(p))
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}
	sentinel, err := attributevalue.MarshalMap(slugSentinelItem{
		ID:       slugSentinelID(p.Slug),
		PolicyID: p.ID,
		Kind:     "slug-sentinel",
	})
	if err != nil {
		return fmt.Errorf("marshal slug sentinel: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbtypes.TransactWriteItem{
			{
				Put: &dynamodbtypes.Put{
					TableName:           aws.String(s.policiesTable),
					Item:                primary,
					ConditionExpression: aws.String("attribute_not_exists(id)"),
				},
			},
			{
				Put: &dynamodbtypes.Put{
					TableName:           aws.String(s.policiesTable),
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
		return fmt.Errorf("create policy: %w", err)
	}

	rev := policyRevisionItem{
		PolicyID:  p.ID,
		Revision:  revisionSortKey(p.Revision),
		BodySv:    p.BodySv,
		BodyEn:    p.BodyEn,
		EditedAt:  p.CreatedAt.UTC().Format(time.RFC3339),
		EditedBy:  p.LastEditedBy,
		Published: false,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return fmt.Errorf("marshal policy revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.policyRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return fmt.Errorf("put policy revision: %w", err)
	}
	return nil
}

// UpdatePolicyDraft replaces the bodies on a draft policy. Refuses to touch
// anything that isn't currently in the "draft" status — for published
// policies, callers must go through EditPublishedPolicy (which bumps the
// revision counter and appends history).
func (s *DynamoStore) UpdatePolicyDraft(ctx context.Context, id, bodySv, bodyEn, editor string) error {
	if bodySv == "" || bodyEn == "" {
		return errors.New("policy bodies required (sv + en)")
	}
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.policiesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET bodySv = :sv, bodyEn = :en, updatedAt = :u, lastEditedBy = :e"),
		ConditionExpression: aws.String("attribute_exists(id) AND #status = :draft"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":sv":    &dynamodbtypes.AttributeValueMemberS{Value: bodySv},
			":en":    &dynamodbtypes.AttributeValueMemberS{Value: bodyEn},
			":u":     &dynamodbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			":e":     &dynamodbtypes.AttributeValueMemberS{Value: editor},
			":draft": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusDraft)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			// Could be 404 or wrong status. Look up to disambiguate.
			existing, lookupErr := s.GetPolicyByID(ctx, id)
			if lookupErr != nil {
				return lookupErr
			}
			if existing == nil {
				return ErrNotFound
			}
			return ErrInvalidPolicyState
		}
		return fmt.Errorf("update policy draft: %w", err)
	}

	// Drafts have a single revision (#1); overwrite its body so the history
	// row matches the head. This keeps GetPolicyRevision consistent for the
	// draft's revision 1 if anyone ever asks for it.
	rev := policyRevisionItem{
		PolicyID: id,
		Revision: revisionSortKey(1),
		BodySv:   bodySv,
		BodyEn:   bodyEn,
		EditedAt: time.Now().UTC().Format(time.RFC3339),
		EditedBy: editor,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return fmt.Errorf("marshal policy revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.policyRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return fmt.Errorf("put policy revision: %w", err)
	}
	return nil
}

// EditPublishedPolicy bumps the revision counter on a published policy and
// appends a new row in the revisions history table. The previous body
// remains queryable via GetPolicyRevision, so runners who consented under
// an older revision can always see what they accepted.
//
// Returns ErrInvalidPolicyState if the policy isn't currently published.
func (s *DynamoStore) EditPublishedPolicy(ctx context.Context, id, bodySv, bodyEn, editor, note string) (int, error) {
	if bodySv == "" || bodyEn == "" {
		return 0, errors.New("policy bodies required (sv + en)")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.policiesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET bodySv = :sv, bodyEn = :en, updatedAt = :u, lastEditedBy = :e, revision = revision + :one"),
		ConditionExpression: aws.String("attribute_exists(id) AND #status = :pub"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":sv":  &dynamodbtypes.AttributeValueMemberS{Value: bodySv},
			":en":  &dynamodbtypes.AttributeValueMemberS{Value: bodyEn},
			":u":   &dynamodbtypes.AttributeValueMemberS{Value: now},
			":e":   &dynamodbtypes.AttributeValueMemberS{Value: editor},
			":one": &dynamodbtypes.AttributeValueMemberN{Value: "1"},
			":pub": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusPublished)},
		},
		ReturnValues: dynamodbtypes.ReturnValueAllNew,
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetPolicyByID(ctx, id)
			if lookupErr != nil {
				return 0, lookupErr
			}
			if existing == nil {
				return 0, ErrNotFound
			}
			return 0, ErrInvalidPolicyState
		}
		return 0, fmt.Errorf("edit published policy: %w", err)
	}

	var updated policyItem
	if err := attributevalue.UnmarshalMap(out.Attributes, &updated); err != nil {
		return 0, fmt.Errorf("unmarshal updated policy: %w", err)
	}

	rev := policyRevisionItem{
		PolicyID: id,
		Revision: revisionSortKey(updated.Revision),
		BodySv:   bodySv,
		BodyEn:   bodyEn,
		EditedAt: now,
		EditedBy: editor,
		Note:     note,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return 0, fmt.Errorf("marshal policy revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.policyRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return 0, fmt.Errorf("put policy revision: %w", err)
	}
	return updated.Revision, nil
}

// PublishPolicy flips a draft to published. If another policy is already
// published, this archives it in the same TransactWriteItems call —
// guaranteeing exactly-one-published. Returns ErrInvalidPolicyState if the
// target isn't currently a draft.
func (s *DynamoStore) PublishPolicy(ctx context.Context, id, publisherID string, at time.Time) error {
	target, err := s.GetPolicyByID(ctx, id)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotFound
	}
	if target.Status != models.PolicyStatusDraft {
		return ErrInvalidPolicyState
	}
	if target.BodySv == "" || target.BodyEn == "" {
		return errors.New("policy bodies required before publishing")
	}

	atStr := at.UTC().Format(time.RFC3339)
	transactItems := []dynamodbtypes.TransactWriteItem{}

	// Archive the currently-published policy (if any). Without this, two
	// policies could end up in "published" status.
	current, err := s.GetPublishedPolicy(ctx)
	if err != nil && !errors.Is(err, ErrNoPublishedPolicy) {
		return err
	}
	if current != nil && current.ID != id {
		transactItems = append(transactItems, dynamodbtypes.TransactWriteItem{
			Update: &dynamodbtypes.Update{
				TableName: aws.String(s.policiesTable),
				Key: map[string]dynamodbtypes.AttributeValue{
					"id": &dynamodbtypes.AttributeValueMemberS{Value: current.ID},
				},
				UpdateExpression:    aws.String("SET #status = :arch, updatedAt = :u"),
				ConditionExpression: aws.String("#status = :pub"),
				ExpressionAttributeNames: map[string]string{
					"#status": "status",
				},
				ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
					":arch": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusArchived)},
					":pub":  &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusPublished)},
					":u":    &dynamodbtypes.AttributeValueMemberS{Value: atStr},
				},
			},
		})
	}

	// Flip the target draft → published.
	transactItems = append(transactItems, dynamodbtypes.TransactWriteItem{
		Update: &dynamodbtypes.Update{
			TableName: aws.String(s.policiesTable),
			Key: map[string]dynamodbtypes.AttributeValue{
				"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
			},
			UpdateExpression:    aws.String("SET #status = :pub, publishedAt = :u, publishedBy = :pby, updatedAt = :u"),
			ConditionExpression: aws.String("#status = :draft"),
			ExpressionAttributeNames: map[string]string{
				"#status": "status",
			},
			ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
				":pub":   &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusPublished)},
				":draft": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusDraft)},
				":u":     &dynamodbtypes.AttributeValueMemberS{Value: atStr},
				":pby":   &dynamodbtypes.AttributeValueMemberS{Value: publisherID},
			},
		},
	})

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		if isTransactionCanceled(err) {
			return ErrInvalidPolicyState
		}
		return fmt.Errorf("publish policy: %w", err)
	}

	// Mark the revision that's now live in the snapshot history.
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.policyRevisionsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"policyId": &dynamodbtypes.AttributeValueMemberS{Value: id},
			"revision": &dynamodbtypes.AttributeValueMemberS{Value: revisionSortKey(target.Revision)},
		},
		UpdateExpression: aws.String("SET published = :t"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":t": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
	})
	if err != nil {
		// Best-effort marker; not fatal.
		return nil
	}
	return nil
}

// ArchivePolicy flips a published policy to archived. Used when retiring a
// policy without a replacement (rare — usually a new publish handles this
// atomically). Returns ErrInvalidPolicyState if the target isn't currently
// published.
func (s *DynamoStore) ArchivePolicy(ctx context.Context, id string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.policiesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :arch, updatedAt = :u"),
		ConditionExpression: aws.String("attribute_exists(id) AND #status = :pub"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":arch": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusArchived)},
			":pub":  &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusPublished)},
			":u":    &dynamodbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetPolicyByID(ctx, id)
			if lookupErr != nil {
				return lookupErr
			}
			if existing == nil {
				return ErrNotFound
			}
			return ErrInvalidPolicyState
		}
		return fmt.Errorf("archive policy: %w", err)
	}
	return nil
}

func (s *DynamoStore) GetPolicyByID(ctx context.Context, id string) (*models.Policy, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.policiesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}
	var item policyItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	p := policyFromItem(item)
	return &p, nil
}

func (s *DynamoStore) GetPolicyBySlug(ctx context.Context, slug string) (*models.Policy, error) {
	sentOut, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.policiesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: slugSentinelID(slug)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get slug sentinel: %w", err)
	}
	if sentOut.Item == nil {
		return nil, nil
	}
	var sentinel slugSentinelItem
	if err := attributevalue.UnmarshalMap(sentOut.Item, &sentinel); err != nil {
		return nil, fmt.Errorf("unmarshal slug sentinel: %w", err)
	}
	if sentinel.PolicyID == "" {
		return nil, errors.New("slug sentinel row is missing policyId")
	}
	return s.GetPolicyByID(ctx, sentinel.PolicyID)
}

// GetPublishedPolicy returns the single policy currently in "published"
// status. Returns ErrNoPublishedPolicy when there isn't one. Uses the
// byStatus GSI for an O(1)-ish lookup; if more than one row matches, the
// schema invariant is broken and the first match wins (best effort).
func (s *DynamoStore) GetPublishedPolicy(ctx context.Context) (*models.Policy, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.policiesTable),
		IndexName:              aws.String(byStatusIndex),
		KeyConditionExpression: aws.String("#status = :pub"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pub": &dynamodbtypes.AttributeValueMemberS{Value: string(models.PolicyStatusPublished)},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query published policy: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, ErrNoPublishedPolicy
	}
	var item policyItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	p := policyFromItem(item)
	return &p, nil
}

// ListPolicies returns every primary policy row (excludes SLUG# sentinels by
// filtering on the `kind` attribute) sorted by CreatedAt descending.
func (s *DynamoStore) ListPolicies(ctx context.Context) ([]models.Policy, error) {
	items, err := s.scanAll(ctx, s.policiesTable)
	if err != nil {
		return nil, fmt.Errorf("scan policies: %w", err)
	}
	policies := make([]models.Policy, 0, len(items))
	for _, raw := range items {
		var it policyItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			continue
		}
		if it.Kind != "policy" {
			continue
		}
		policies = append(policies, policyFromItem(it))
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].CreatedAt.After(policies[j].CreatedAt)
	})
	return policies, nil
}

func (s *DynamoStore) GetPolicyRevision(ctx context.Context, policyID string, revision int) (*models.PolicyRevision, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.policyRevisionsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"policyId": &dynamodbtypes.AttributeValueMemberS{Value: policyID},
			"revision": &dynamodbtypes.AttributeValueMemberS{Value: revisionSortKey(revision)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get policy revision: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item policyRevisionItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal policy revision: %w", err)
	}
	r := policyRevisionFromItem(item)
	return &r, nil
}

// ListPolicyRevisions returns every revision row for a policy, newest first.
func (s *DynamoStore) ListPolicyRevisions(ctx context.Context, policyID string) ([]models.PolicyRevision, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.policyRevisionsTable),
		KeyConditionExpression: aws.String("policyId = :p"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":p": &dynamodbtypes.AttributeValueMemberS{Value: policyID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("query policy revisions: %w", err)
	}
	revisions := make([]models.PolicyRevision, 0, len(out.Items))
	for _, raw := range out.Items {
		var item policyRevisionItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal policy revision: %w", err)
		}
		revisions = append(revisions, policyRevisionFromItem(item))
	}
	return revisions, nil
}
