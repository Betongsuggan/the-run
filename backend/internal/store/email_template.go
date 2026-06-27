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

// emailTemplateItem is the on-the-wire shape. The slug doubles as the primary
// key — the set of valid slugs is closed in models.KnownEmailTemplateSlugs, so
// there's no sentinel-row dance and no UUIDs.
type emailTemplateItem struct {
	Slug               string   `dynamodbav:"slug"`
	DisplayName        string   `dynamodbav:"displayName"`
	Status             string   `dynamodbav:"status"`
	Revision           int      `dynamodbav:"revision"`
	SubjectSv          string   `dynamodbav:"subjectSv"`
	BodySv             string   `dynamodbav:"bodySv"`
	SubjectEn          string   `dynamodbav:"subjectEn"`
	BodyEn             string   `dynamodbav:"bodyEn"`
	AvailableVariables []string `dynamodbav:"availableVariables,omitempty"`
	CreatedAt          string   `dynamodbav:"createdAt"`
	UpdatedAt          string   `dynamodbav:"updatedAt"`
	PublishedAt        string   `dynamodbav:"publishedAt,omitempty"`
	PublishedBy        string   `dynamodbav:"publishedBy,omitempty"`
	LastEditedBy       string   `dynamodbav:"lastEditedBy,omitempty"`
}

type emailTemplateRevisionItem struct {
	Slug      string `dynamodbav:"slug"`
	Revision  string `dynamodbav:"revision"` // zero-padded for lex order
	SubjectSv string `dynamodbav:"subjectSv"`
	BodySv    string `dynamodbav:"bodySv"`
	SubjectEn string `dynamodbav:"subjectEn"`
	BodyEn    string `dynamodbav:"bodyEn"`
	EditedAt  string `dynamodbav:"editedAt"`
	EditedBy  string `dynamodbav:"editedBy,omitempty"`
	Note      string `dynamodbav:"note,omitempty"`
	Published bool   `dynamodbav:"published"`
}

func emailTemplateFromItem(item emailTemplateItem) models.EmailTemplate {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, item.UpdatedAt)
	t := models.EmailTemplate{
		Slug:               models.EmailTemplateSlug(item.Slug),
		DisplayName:        item.DisplayName,
		Status:             models.EmailTemplateStatus(item.Status),
		Revision:           item.Revision,
		SubjectSv:          item.SubjectSv,
		BodySv:             item.BodySv,
		SubjectEn:          item.SubjectEn,
		BodyEn:             item.BodyEn,
		AvailableVariables: item.AvailableVariables,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		PublishedBy:        item.PublishedBy,
		LastEditedBy:       item.LastEditedBy,
	}
	if item.PublishedAt != "" {
		if pt, err := time.Parse(time.RFC3339, item.PublishedAt); err == nil {
			t.PublishedAt = &pt
		}
	}
	return t
}

func itemFromEmailTemplate(t models.EmailTemplate) emailTemplateItem {
	item := emailTemplateItem{
		Slug:               string(t.Slug),
		DisplayName:        t.DisplayName,
		Status:             string(t.Status),
		Revision:           t.Revision,
		SubjectSv:          t.SubjectSv,
		BodySv:             t.BodySv,
		SubjectEn:          t.SubjectEn,
		BodyEn:             t.BodyEn,
		AvailableVariables: t.AvailableVariables,
		CreatedAt:          t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.UTC().Format(time.RFC3339),
		PublishedBy:        t.PublishedBy,
		LastEditedBy:       t.LastEditedBy,
	}
	if t.PublishedAt != nil {
		item.PublishedAt = t.PublishedAt.UTC().Format(time.RFC3339)
	}
	return item
}

func emailTemplateRevisionFromItem(item emailTemplateRevisionItem) models.EmailTemplateRevision {
	editedAt, _ := time.Parse(time.RFC3339, item.EditedAt)
	var revision int
	fmt.Sscanf(item.Revision, "%d", &revision)
	return models.EmailTemplateRevision{
		Slug:      models.EmailTemplateSlug(item.Slug),
		Revision:  revision,
		SubjectSv: item.SubjectSv,
		BodySv:    item.BodySv,
		SubjectEn: item.SubjectEn,
		BodyEn:    item.BodyEn,
		EditedAt:  editedAt,
		EditedBy:  item.EditedBy,
		Note:      item.Note,
		Published: item.Published,
	}
}

// CreateEmailTemplate writes the head row and revision 1 of a new template.
// Called only by the seed routine on first boot; admins never create
// templates via the API (the set of slugs is fixed in code).
// Returns ErrAlreadyExists if a template with the same slug already exists.
func (s *DynamoStore) CreateEmailTemplate(ctx context.Context, t models.EmailTemplate) error {
	if t.Slug == "" {
		return errors.New("email template slug required")
	}
	if !models.IsKnownEmailTemplateSlug(string(t.Slug)) {
		return fmt.Errorf("email template slug %q is not in the known set", t.Slug)
	}
	if t.Status == "" {
		t.Status = models.EmailTemplateStatusDraft
	}
	if t.Revision == 0 {
		t.Revision = 1
	}
	primary, err := attributevalue.MarshalMap(itemFromEmailTemplate(t))
	if err != nil {
		return fmt.Errorf("marshal email template: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.emailTemplatesTable),
		Item:                primary,
		ConditionExpression: aws.String("attribute_not_exists(slug)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("create email template: %w", err)
	}
	rev := emailTemplateRevisionItem{
		Slug:      string(t.Slug),
		Revision:  revisionSortKey(t.Revision),
		SubjectSv: t.SubjectSv,
		BodySv:    t.BodySv,
		SubjectEn: t.SubjectEn,
		BodyEn:    t.BodyEn,
		EditedAt:  t.CreatedAt.UTC().Format(time.RFC3339),
		EditedBy:  t.LastEditedBy,
		Published: t.Status == models.EmailTemplateStatusPublished,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return fmt.Errorf("marshal email template revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.emailTemplateRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return fmt.Errorf("put email template revision: %w", err)
	}
	return nil
}

// UpdateEmailTemplateDraft replaces subject + body on a draft template. Refuses
// to touch anything that isn't currently in "draft" — use EditPublishedEmail
// Template for published rows so the revision counter bumps and history is
// appended.
func (s *DynamoStore) UpdateEmailTemplateDraft(ctx context.Context, slug models.EmailTemplateSlug, subjectSv, bodySv, subjectEn, bodyEn, editor string) error {
	if subjectSv == "" || bodySv == "" || subjectEn == "" || bodyEn == "" {
		return errors.New("subject and body required for both languages")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.emailTemplatesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
		UpdateExpression:    aws.String("SET subjectSv = :ss, bodySv = :bs, subjectEn = :se, bodyEn = :be, updatedAt = :u, lastEditedBy = :e"),
		ConditionExpression: aws.String("attribute_exists(slug) AND #status = :draft"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":ss":    &dynamodbtypes.AttributeValueMemberS{Value: subjectSv},
			":bs":    &dynamodbtypes.AttributeValueMemberS{Value: bodySv},
			":se":    &dynamodbtypes.AttributeValueMemberS{Value: subjectEn},
			":be":    &dynamodbtypes.AttributeValueMemberS{Value: bodyEn},
			":u":     &dynamodbtypes.AttributeValueMemberS{Value: now},
			":e":     &dynamodbtypes.AttributeValueMemberS{Value: editor},
			":draft": &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusDraft)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetEmailTemplate(ctx, slug)
			if lookupErr != nil {
				return lookupErr
			}
			if existing == nil {
				return ErrNotFound
			}
			return ErrInvalidEmailTemplateState
		}
		return fmt.Errorf("update email template draft: %w", err)
	}

	rev := emailTemplateRevisionItem{
		Slug:      string(slug),
		Revision:  revisionSortKey(1),
		SubjectSv: subjectSv,
		BodySv:    bodySv,
		SubjectEn: subjectEn,
		BodyEn:    bodyEn,
		EditedAt:  now,
		EditedBy:  editor,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return fmt.Errorf("marshal email template revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.emailTemplateRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return fmt.Errorf("put email template revision: %w", err)
	}
	return nil
}

// EditPublishedEmailTemplate is the published-template edit path: bumps the
// revision counter and appends a snapshot row. Returns ErrInvalidEmail
// TemplateState if the row isn't currently published.
func (s *DynamoStore) EditPublishedEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug, subjectSv, bodySv, subjectEn, bodyEn, editor, note string) (int, error) {
	if subjectSv == "" || bodySv == "" || subjectEn == "" || bodyEn == "" {
		return 0, errors.New("subject and body required for both languages")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.emailTemplatesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
		UpdateExpression:    aws.String("SET subjectSv = :ss, bodySv = :bs, subjectEn = :se, bodyEn = :be, updatedAt = :u, lastEditedBy = :e, revision = revision + :one"),
		ConditionExpression: aws.String("attribute_exists(slug) AND #status = :pub"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":ss":  &dynamodbtypes.AttributeValueMemberS{Value: subjectSv},
			":bs":  &dynamodbtypes.AttributeValueMemberS{Value: bodySv},
			":se":  &dynamodbtypes.AttributeValueMemberS{Value: subjectEn},
			":be":  &dynamodbtypes.AttributeValueMemberS{Value: bodyEn},
			":u":   &dynamodbtypes.AttributeValueMemberS{Value: now},
			":e":   &dynamodbtypes.AttributeValueMemberS{Value: editor},
			":one": &dynamodbtypes.AttributeValueMemberN{Value: "1"},
			":pub": &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusPublished)},
		},
		ReturnValues: dynamodbtypes.ReturnValueAllNew,
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetEmailTemplate(ctx, slug)
			if lookupErr != nil {
				return 0, lookupErr
			}
			if existing == nil {
				return 0, ErrNotFound
			}
			return 0, ErrInvalidEmailTemplateState
		}
		return 0, fmt.Errorf("edit published email template: %w", err)
	}
	var updated emailTemplateItem
	if err := attributevalue.UnmarshalMap(out.Attributes, &updated); err != nil {
		return 0, fmt.Errorf("unmarshal updated email template: %w", err)
	}

	rev := emailTemplateRevisionItem{
		Slug:      string(slug),
		Revision:  revisionSortKey(updated.Revision),
		SubjectSv: subjectSv,
		BodySv:    bodySv,
		SubjectEn: subjectEn,
		BodyEn:    bodyEn,
		EditedAt:  now,
		EditedBy:  editor,
		Note:      note,
		Published: true,
	}
	revAV, err := attributevalue.MarshalMap(rev)
	if err != nil {
		return 0, fmt.Errorf("marshal email template revision: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.emailTemplateRevisionsTable),
		Item:      revAV,
	}); err != nil {
		return 0, fmt.Errorf("put email template revision: %w", err)
	}
	return updated.Revision, nil
}

// PublishEmailTemplate flips a draft to published. Unlike policies, each
// template's slug already uniquely identifies it — there's no
// archive-the-previous-one step.
func (s *DynamoStore) PublishEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug, publisher string, at time.Time) error {
	atStr := at.UTC().Format(time.RFC3339)
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.emailTemplatesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
		UpdateExpression:    aws.String("SET #status = :pub, publishedAt = :u, publishedBy = :pby, updatedAt = :u"),
		ConditionExpression: aws.String("attribute_exists(slug) AND #status = :draft"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pub":   &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusPublished)},
			":draft": &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusDraft)},
			":u":     &dynamodbtypes.AttributeValueMemberS{Value: atStr},
			":pby":   &dynamodbtypes.AttributeValueMemberS{Value: publisher},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetEmailTemplate(ctx, slug)
			if lookupErr != nil {
				return lookupErr
			}
			if existing == nil {
				return ErrNotFound
			}
			return ErrInvalidEmailTemplateState
		}
		return fmt.Errorf("publish email template: %w", err)
	}

	// Mark the snapshot row as the one that's now live. Best-effort.
	template, err := s.GetEmailTemplate(ctx, slug)
	if err != nil || template == nil {
		return nil
	}
	_, _ = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.emailTemplateRevisionsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug":     &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
			"revision": &dynamodbtypes.AttributeValueMemberS{Value: revisionSortKey(template.Revision)},
		},
		UpdateExpression: aws.String("SET published = :t"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":t": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
	})
	return nil
}

// ArchiveEmailTemplate flips a published template back to archived. Renderer
// callers should refuse to send when the only matching record is archived.
func (s *DynamoStore) ArchiveEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.emailTemplatesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
		UpdateExpression:    aws.String("SET #status = :arch, updatedAt = :u"),
		ConditionExpression: aws.String("attribute_exists(slug) AND #status = :pub"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":arch": &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusArchived)},
			":pub":  &dynamodbtypes.AttributeValueMemberS{Value: string(models.EmailTemplateStatusPublished)},
			":u":    &dynamodbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			existing, lookupErr := s.GetEmailTemplate(ctx, slug)
			if lookupErr != nil {
				return lookupErr
			}
			if existing == nil {
				return ErrNotFound
			}
			return ErrInvalidEmailTemplateState
		}
		return fmt.Errorf("archive email template: %w", err)
	}
	return nil
}

func (s *DynamoStore) GetEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug) (*models.EmailTemplate, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.emailTemplatesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"slug": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get email template: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}
	var item emailTemplateItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal email template: %w", err)
	}
	t := emailTemplateFromItem(item)
	return &t, nil
}

func (s *DynamoStore) ListEmailTemplates(ctx context.Context) ([]models.EmailTemplate, error) {
	items, err := s.scanAll(ctx, s.emailTemplatesTable)
	if err != nil {
		return nil, fmt.Errorf("scan email templates: %w", err)
	}
	templates := make([]models.EmailTemplate, 0, len(items))
	for _, raw := range items {
		var it emailTemplateItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			continue
		}
		templates = append(templates, emailTemplateFromItem(it))
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Slug < templates[j].Slug
	})
	return templates, nil
}

func (s *DynamoStore) ListEmailTemplateRevisions(ctx context.Context, slug models.EmailTemplateSlug) ([]models.EmailTemplateRevision, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.emailTemplateRevisionsTable),
		KeyConditionExpression: aws.String("slug = :s"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":s": &dynamodbtypes.AttributeValueMemberS{Value: string(slug)},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("query email template revisions: %w", err)
	}
	revisions := make([]models.EmailTemplateRevision, 0, len(out.Items))
	for _, raw := range out.Items {
		var it emailTemplateRevisionItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			return nil, fmt.Errorf("unmarshal email template revision: %w", err)
		}
		revisions = append(revisions, emailTemplateRevisionFromItem(it))
	}
	return revisions, nil
}
