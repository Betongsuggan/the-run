package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

const (
	byNameDOBIndex = "byNameDOB"
	byAccountIndex = "byAccount"
	byRunnerIndex  = "byRunner"
	byEventIndex   = "byEvent"
	byStatusIndex  = "byStatus"
)

type DynamoStore struct {
	client                      *dynamodb.Client
	runnersTable                string
	registrationsTable          string
	eventsTable                 string
	racesTable                  string
	accountsTable               string
	authAttemptsTable           string
	magicTokensTable            string
	rateLimitTable              string
	policiesTable               string
	policyRevisionsTable        string
	emailTemplatesTable         string
	emailTemplateRevisionsTable string
	adminInvitationsTable       string
}

// NewDynamoStoreFromEnv reads RUNNERS_TABLE_NAME, REGISTRATIONS_TABLE_NAME,
// EVENTS_TABLE_NAME, and RACES_TABLE_NAME from the environment, loads the AWS
// SDK's default config, and (if AWS_ENDPOINT_URL is set) points the DynamoDB
// client at it — that's how we target LocalStack in local dev.
func NewDynamoStoreFromEnv(ctx context.Context) (*DynamoStore, error) {
	runnersTable := os.Getenv("RUNNERS_TABLE_NAME")
	registrationsTable := os.Getenv("REGISTRATIONS_TABLE_NAME")
	eventsTable := os.Getenv("EVENTS_TABLE_NAME")
	racesTable := os.Getenv("RACES_TABLE_NAME")
	accountsTable := os.Getenv("ACCOUNTS_TABLE_NAME")
	authAttemptsTable := os.Getenv("AUTH_ATTEMPTS_TABLE_NAME")
	magicTokensTable := os.Getenv("MAGIC_TOKENS_TABLE_NAME")
	rateLimitTable := os.Getenv("RATE_LIMIT_TABLE_NAME")
	policiesTable := os.Getenv("POLICIES_TABLE_NAME")
	policyRevisionsTable := os.Getenv("POLICY_REVISIONS_TABLE_NAME")
	emailTemplatesTable := os.Getenv("EMAIL_TEMPLATES_TABLE_NAME")
	emailTemplateRevisionsTable := os.Getenv("EMAIL_TEMPLATE_REVISIONS_TABLE_NAME")
	adminInvitationsTable := os.Getenv("ADMIN_INVITATIONS_TABLE_NAME")
	if runnersTable == "" || registrationsTable == "" || eventsTable == "" || racesTable == "" || accountsTable == "" || authAttemptsTable == "" || magicTokensTable == "" || rateLimitTable == "" || policiesTable == "" || policyRevisionsTable == "" || emailTemplatesTable == "" || emailTemplateRevisionsTable == "" || adminInvitationsTable == "" {
		return nil, errors.New("RUNNERS_TABLE_NAME, REGISTRATIONS_TABLE_NAME, EVENTS_TABLE_NAME, RACES_TABLE_NAME, ACCOUNTS_TABLE_NAME, AUTH_ATTEMPTS_TABLE_NAME, MAGIC_TOKENS_TABLE_NAME, RATE_LIMIT_TABLE_NAME, POLICIES_TABLE_NAME, POLICY_REVISIONS_TABLE_NAME, EMAIL_TEMPLATES_TABLE_NAME, EMAIL_TEMPLATE_REVISIONS_TABLE_NAME, ADMIN_INVITATIONS_TABLE_NAME must all be set")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var optFns []func(*dynamodb.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		optFns = append(optFns, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &DynamoStore{
		client:                      dynamodb.NewFromConfig(cfg, optFns...),
		runnersTable:                runnersTable,
		registrationsTable:          registrationsTable,
		eventsTable:                 eventsTable,
		racesTable:                  racesTable,
		accountsTable:               accountsTable,
		authAttemptsTable:           authAttemptsTable,
		magicTokensTable:            magicTokensTable,
		rateLimitTable:              rateLimitTable,
		policiesTable:               policiesTable,
		policyRevisionsTable:        policyRevisionsTable,
		emailTemplatesTable:         emailTemplatesTable,
		emailTemplateRevisionsTable: emailTemplateRevisionsTable,
		adminInvitationsTable:       adminInvitationsTable,
	}, nil
}

// ─── Runners ──────────────────────────────────────────────────────────

type runnerItem struct {
	ID                   string       `dynamodbav:"id"`
	AccountID            string       `dynamodbav:"accountId"`
	NameDobKey           string       `dynamodbav:"nameDobKey"`
	Name                 string       `dynamodbav:"name"`
	BirthDate            string       `dynamodbav:"birthDate"`
	Gender               string       `dynamodbav:"gender"`
	PublicResults        *consentItem `dynamodbav:"publicResults,omitempty"`
	CreatedAt            string       `dynamodbav:"createdAt"`
	DeletionPendingUntil string       `dynamodbav:"deletionPendingUntil,omitempty"`
}

func runnerFromItem(item runnerItem) models.Runner {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	r := models.Runner{
		ID:        item.ID,
		AccountID: item.AccountID,
		Name:      item.Name,
		BirthDate: item.BirthDate,
		Gender:    item.Gender,
		CreatedAt: createdAt,
	}
	if item.PublicResults != nil {
		at, _ := time.Parse(time.RFC3339, item.PublicResults.At)
		r.Consents.PublicResults = models.Consent{
			Granted:        item.PublicResults.Granted,
			At:             at,
			PolicyID:       item.PublicResults.PolicyID,
			PolicyRevision: item.PublicResults.PolicyRevision,
			PolicyVersion:  item.PublicResults.PolicyVersion,
		}
	}
	if item.DeletionPendingUntil != "" {
		t, err := time.Parse(time.RFC3339, item.DeletionPendingUntil)
		if err == nil {
			r.DeletionPendingUntil = &t
		}
	}
	return r
}

func itemFromRunner(r models.Runner) runnerItem {
	item := runnerItem{
		ID:         r.ID,
		AccountID:  r.AccountID,
		NameDobKey: r.NameDobKey(),
		Name:       r.Name,
		BirthDate:  r.BirthDate,
		Gender:     r.Gender,
		CreatedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if !r.Consents.PublicResults.At.IsZero() || r.Consents.PublicResults.PolicyVersion != "" {
		item.PublicResults = &consentItem{
			Granted:        r.Consents.PublicResults.Granted,
			At:             r.Consents.PublicResults.At.UTC().Format(time.RFC3339),
			PolicyID:       r.Consents.PublicResults.PolicyID,
			PolicyRevision: r.Consents.PublicResults.PolicyRevision,
			PolicyVersion:  r.Consents.PublicResults.PolicyVersion,
		}
	}
	if r.DeletionPendingUntil != nil {
		item.DeletionPendingUntil = r.DeletionPendingUntil.UTC().Format(time.RFC3339)
	}
	return item
}

// RunnerByNameDOB returns every Runner record matching the given name+DOB key.
// With the Account model, two different Accounts can each own a Runner with
// the same name+DOB; callers filter the returned slice by AccountID.
func (s *DynamoStore) RunnerByNameDOB(ctx context.Context, nameDobKey string) ([]models.Runner, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.runnersTable),
		IndexName:              aws.String(byNameDOBIndex),
		KeyConditionExpression: aws.String("nameDobKey = :k"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":k": &dynamodbtypes.AttributeValueMemberS{Value: nameDobKey},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query runners byNameDOB: %w", err)
	}
	runners := make([]models.Runner, 0, len(out.Items))
	for _, raw := range out.Items {
		var item runnerItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal runner: %w", err)
		}
		runners = append(runners, runnerFromItem(item))
	}
	return runners, nil
}

func (s *DynamoStore) ListRunners(ctx context.Context) ([]models.Runner, error) {
	items, err := s.scanAll(ctx, s.runnersTable)
	if err != nil {
		return nil, fmt.Errorf("scan runners: %w", err)
	}
	runners := make([]models.Runner, 0, len(items))
	for _, raw := range items {
		var item runnerItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal runner: %w", err)
		}
		runners = append(runners, runnerFromItem(item))
	}
	return runners, nil
}

// ListRunnersByAccount returns every Runner owned by a given Account. Used by
// the DSR self-service surface so a guardian can manage themselves + kids in
// one session. Queries the byAccount GSI.
func (s *DynamoStore) ListRunnersByAccount(ctx context.Context, accountID string) ([]models.Runner, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.runnersTable),
		IndexName:              aws.String(byAccountIndex),
		KeyConditionExpression: aws.String("accountId = :a"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":a": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query runners byAccount: %w", err)
	}
	runners := make([]models.Runner, 0, len(out.Items))
	for _, raw := range out.Items {
		var item runnerItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal runner: %w", err)
		}
		runners = append(runners, runnerFromItem(item))
	}
	return runners, nil
}

func (s *DynamoStore) GetRunner(ctx context.Context, id string) (*models.Runner, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.runnersTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get runner: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item runnerItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal runner: %w", err)
	}
	r := runnerFromItem(item)
	return &r, nil
}

func (s *DynamoStore) CreateRunner(ctx context.Context, r models.Runner) error {
	item, err := attributevalue.MarshalMap(itemFromRunner(r))
	if err != nil {
		return fmt.Errorf("marshal runner: %w", err)
	}
	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.runnersTable),
		Item:      item,
	}); err != nil {
		return fmt.Errorf("put runner: %w", err)
	}
	return nil
}

func (s *DynamoStore) UpdateRunner(ctx context.Context, r models.Runner) error {
	item, err := attributevalue.MarshalMap(itemFromRunner(r))
	if err != nil {
		return fmt.Errorf("marshal runner: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.runnersTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("update runner: %w", err)
	}
	return nil
}

func (s *DynamoStore) DeleteRunner(ctx context.Context, id string) error {
	regs, err := s.ListRegistrationsByRunner(ctx, id)
	if err != nil {
		return fmt.Errorf("list registrations for runner: %w", err)
	}
	for _, reg := range regs {
		if err := s.DeleteRegistration(ctx, reg.RaceID, reg.RunnerID); err != nil && !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("cascade delete registration: %w", err)
		}
	}
	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.runnersTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete runner: %w", err)
	}
	return nil
}

// ─── Events ───────────────────────────────────────────────────────────

type eventItem struct {
	ID        string `dynamodbav:"id"`
	Name      string `dynamodbav:"name"`
	Year      int    `dynamodbav:"year"`
	Date      string `dynamodbav:"date"`
	Location  string `dynamodbav:"location,omitempty"`
	CreatedAt string `dynamodbav:"createdAt"`
}

func eventFromItem(item eventItem) models.Event {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	return models.Event{
		ID:        item.ID,
		Name:      item.Name,
		Year:      item.Year,
		Date:      item.Date,
		Location:  item.Location,
		CreatedAt: createdAt,
	}
}

func (s *DynamoStore) ListEvents(ctx context.Context) ([]models.Event, error) {
	items, err := s.scanAll(ctx, s.eventsTable)
	if err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}
	out := make([]models.Event, 0, len(items))
	for _, raw := range items {
		var item eventItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		out = append(out, eventFromItem(item))
	}
	return out, nil
}

func (s *DynamoStore) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.eventsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item eventItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}
	e := eventFromItem(item)
	return &e, nil
}

func (s *DynamoStore) CreateEvent(ctx context.Context, e models.Event) error {
	item, err := attributevalue.MarshalMap(eventItem{
		ID:        e.ID,
		Name:      e.Name,
		Year:      e.Year,
		Date:      e.Date,
		Location:  e.Location,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.eventsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("put event: %w", err)
	}
	return nil
}

func (s *DynamoStore) UpdateEvent(ctx context.Context, e models.Event) error {
	item, err := attributevalue.MarshalMap(eventItem{
		ID:        e.ID,
		Name:      e.Name,
		Year:      e.Year,
		Date:      e.Date,
		Location:  e.Location,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.eventsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (s *DynamoStore) DeleteEvent(ctx context.Context, id string) error {
	races, err := s.ListRacesByEvent(ctx, id)
	if err != nil {
		return fmt.Errorf("list races for event: %w", err)
	}
	for _, r := range races {
		if err := s.DeleteRace(ctx, r.ID); err != nil && !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("cascade delete race: %w", err)
		}
	}
	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.eventsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

// ─── Races ────────────────────────────────────────────────────────────

type raceItem struct {
	ID                 string `dynamodbav:"id"`
	EventID            string `dynamodbav:"eventId"`
	Name               string `dynamodbav:"name"`
	DistanceMeters     int    `dynamodbav:"distanceMeters"`
	Discipline         string `dynamodbav:"discipline"`
	MaxRunners         int    `dynamodbav:"maxRunners,omitempty"`
	RegistrationFeeOre int    `dynamodbav:"registrationFeeOre,omitempty"`
	CreatedAt          string `dynamodbav:"createdAt"`
}

func raceFromItem(item raceItem) models.Race {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	return models.Race{
		ID:                 item.ID,
		EventID:            item.EventID,
		Name:               item.Name,
		DistanceMeters:     item.DistanceMeters,
		Discipline:         item.Discipline,
		MaxRunners:         item.MaxRunners,
		RegistrationFeeOre: item.RegistrationFeeOre,
		CreatedAt:          createdAt,
	}
}

func (s *DynamoStore) ListRaces(ctx context.Context) ([]models.Race, error) {
	items, err := s.scanAll(ctx, s.racesTable)
	if err != nil {
		return nil, fmt.Errorf("scan races: %w", err)
	}
	out := make([]models.Race, 0, len(items))
	for _, raw := range items {
		var item raceItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal race: %w", err)
		}
		out = append(out, raceFromItem(item))
	}
	return out, nil
}

func (s *DynamoStore) ListRacesByEvent(ctx context.Context, eventID string) ([]models.Race, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.racesTable),
		IndexName:              aws.String(byEventIndex),
		KeyConditionExpression: aws.String("eventId = :e"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":e": &dynamodbtypes.AttributeValueMemberS{Value: eventID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query races byEvent: %w", err)
	}
	res := make([]models.Race, 0, len(out.Items))
	for _, raw := range out.Items {
		var item raceItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal race: %w", err)
		}
		res = append(res, raceFromItem(item))
	}
	return res, nil
}

func (s *DynamoStore) GetRace(ctx context.Context, id string) (*models.Race, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.racesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get race: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item raceItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal race: %w", err)
	}
	r := raceFromItem(item)
	return &r, nil
}

func (s *DynamoStore) CreateRace(ctx context.Context, r models.Race) error {
	item, err := attributevalue.MarshalMap(raceItem{
		ID:                 r.ID,
		EventID:            r.EventID,
		Name:               r.Name,
		DistanceMeters:     r.DistanceMeters,
		Discipline:         r.Discipline,
		MaxRunners:         r.MaxRunners,
		RegistrationFeeOre: r.RegistrationFeeOre,
		CreatedAt:          r.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal race: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.racesTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("put race: %w", err)
	}
	return nil
}

func (s *DynamoStore) UpdateRace(ctx context.Context, r models.Race) error {
	item, err := attributevalue.MarshalMap(raceItem{
		ID:                 r.ID,
		EventID:            r.EventID,
		Name:               r.Name,
		DistanceMeters:     r.DistanceMeters,
		Discipline:         r.Discipline,
		MaxRunners:         r.MaxRunners,
		RegistrationFeeOre: r.RegistrationFeeOre,
		CreatedAt:          r.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal race: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.racesTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("update race: %w", err)
	}
	return nil
}

func (s *DynamoStore) DeleteRace(ctx context.Context, id string) error {
	regs, err := s.ListRegistrationsByRace(ctx, id)
	if err != nil {
		return fmt.Errorf("list registrations for race: %w", err)
	}
	for _, reg := range regs {
		if err := s.DeleteRegistration(ctx, reg.RaceID, reg.RunnerID); err != nil && !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("cascade delete registration: %w", err)
		}
	}
	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.racesTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete race: %w", err)
	}
	return nil
}

// ─── Registrations ────────────────────────────────────────────────────

type registrationItem struct {
	RaceID            string                  `dynamodbav:"raceId"`
	RunnerID          string                  `dynamodbav:"runnerId"`
	ID                string                  `dynamodbav:"id"`
	Status            string                  `dynamodbav:"status"`
	Bib               string                  `dynamodbav:"bib,omitempty"`
	Category          *registrationCategory   `dynamodbav:"category,omitempty"`
	FinishSeconds     *int                    `dynamodbav:"finishSeconds,omitempty"`
	Splits            []registrationSplitItem `dynamodbav:"splits,omitempty"`
	Conditions        string                  `dynamodbav:"conditions,omitempty"`
	Notes             string                  `dynamodbav:"notes,omitempty"`
	PaymentReceivedAt string                  `dynamodbav:"paymentReceivedAt,omitempty"`
	CreatedAt         string                  `dynamodbav:"createdAt"`
}

type registrationCategory struct {
	Gender   string `dynamodbav:"gender,omitempty"`
	AgeGroup string `dynamodbav:"ageGroup,omitempty"`
}

type registrationSplitItem struct {
	Km          int `dynamodbav:"km"`
	TimeSeconds int `dynamodbav:"timeSeconds"`
}

func registrationFromItem(item registrationItem) models.Registration {
	createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
	reg := models.Registration{
		ID:         item.ID,
		RaceID:     item.RaceID,
		RunnerID:   item.RunnerID,
		Status:     normalizeStatus(item.Status),
		Bib:        item.Bib,
		Conditions: item.Conditions,
		Notes:      item.Notes,
		CreatedAt:  createdAt,
	}
	if item.Category != nil {
		reg.Category = &models.Category{Gender: item.Category.Gender, AgeGroup: item.Category.AgeGroup}
	}
	if item.FinishSeconds != nil {
		v := *item.FinishSeconds
		reg.FinishSeconds = &v
	}
	if len(item.Splits) > 0 {
		reg.Splits = make([]models.Split, len(item.Splits))
		for i, s := range item.Splits {
			reg.Splits[i] = models.Split{Km: s.Km, TimeSeconds: s.TimeSeconds}
		}
	}
	if item.PaymentReceivedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.PaymentReceivedAt); err == nil {
			reg.PaymentReceivedAt = &t
		}
	}
	return reg
}

// normalizeStatus maps the legacy "received" status (written by the public
// POST /registrations endpoint) to the admin vocabulary used by the UI.
func normalizeStatus(s string) string {
	if s == models.StatusReceived {
		return models.StatusPending
	}
	return s
}

func (s *DynamoStore) CreateRegistration(ctx context.Context, reg models.Registration) error {
	item := registrationItem{
		RaceID:    reg.RaceID,
		RunnerID:  reg.RunnerID,
		ID:        reg.ID,
		Status:    reg.Status,
		Bib:       reg.Bib,
		CreatedAt: reg.CreatedAt.UTC().Format(time.RFC3339),
	}
	if reg.Category != nil {
		item.Category = &registrationCategory{Gender: reg.Category.Gender, AgeGroup: reg.Category.AgeGroup}
	}
	if reg.FinishSeconds != nil {
		v := *reg.FinishSeconds
		item.FinishSeconds = &v
	}
	if len(reg.Splits) > 0 {
		item.Splits = make([]registrationSplitItem, len(reg.Splits))
		for i, sp := range reg.Splits {
			item.Splits[i] = registrationSplitItem{Km: sp.Km, TimeSeconds: sp.TimeSeconds}
		}
	}
	if reg.PaymentReceivedAt != nil {
		item.PaymentReceivedAt = reg.PaymentReceivedAt.UTC().Format(time.RFC3339)
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.registrationsTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(raceId) AND attribute_not_exists(runnerId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrAlreadyRegistered
		}
		return fmt.Errorf("put registration: %w", err)
	}
	return nil
}

func (s *DynamoStore) ListRegistrations(ctx context.Context) ([]models.Registration, error) {
	items, err := s.scanAll(ctx, s.registrationsTable)
	if err != nil {
		return nil, fmt.Errorf("scan registrations: %w", err)
	}
	out := make([]models.Registration, 0, len(items))
	for _, raw := range items {
		var item registrationItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal registration: %w", err)
		}
		out = append(out, registrationFromItem(item))
	}
	return out, nil
}

func (s *DynamoStore) ListRegistrationsByRace(ctx context.Context, raceID string) ([]models.Registration, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.registrationsTable),
		KeyConditionExpression: aws.String("raceId = :r"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":r": &dynamodbtypes.AttributeValueMemberS{Value: raceID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query registrations byRace: %w", err)
	}
	res := make([]models.Registration, 0, len(out.Items))
	for _, raw := range out.Items {
		var item registrationItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal registration: %w", err)
		}
		res = append(res, registrationFromItem(item))
	}
	return res, nil
}

func (s *DynamoStore) ListRegistrationsByRunner(ctx context.Context, runnerID string) ([]models.Registration, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.registrationsTable),
		IndexName:              aws.String(byRunnerIndex),
		KeyConditionExpression: aws.String("runnerId = :r"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":r": &dynamodbtypes.AttributeValueMemberS{Value: runnerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query registrations byRunner: %w", err)
	}
	res := make([]models.Registration, 0, len(out.Items))
	for _, raw := range out.Items {
		var item registrationItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal registration: %w", err)
		}
		res = append(res, registrationFromItem(item))
	}
	return res, nil
}

func (s *DynamoStore) GetRegistration(ctx context.Context, raceID, runnerID string) (*models.Registration, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.registrationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"raceId":   &dynamodbtypes.AttributeValueMemberS{Value: raceID},
			"runnerId": &dynamodbtypes.AttributeValueMemberS{Value: runnerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get registration: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var item registrationItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal registration: %w", err)
	}
	reg := registrationFromItem(item)
	return &reg, nil
}

// GetRegistrationByID scans the registrations table. The current schema has
// no byId GSI; this is fine at admin scale but should be revisited if list
// sizes grow.
func (s *DynamoStore) GetRegistrationByID(ctx context.Context, id string) (*models.Registration, error) {
	items, err := s.scanAll(ctx, s.registrationsTable)
	if err != nil {
		return nil, fmt.Errorf("scan registrations: %w", err)
	}
	for _, raw := range items {
		var item registrationItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal registration: %w", err)
		}
		if item.ID == id {
			reg := registrationFromItem(item)
			return &reg, nil
		}
	}
	return nil, ErrNotFound
}

func (s *DynamoStore) UpdateRegistration(ctx context.Context, u RegistrationUpdate) (*models.Registration, error) {
	sets := []string{}
	removes := []string{}
	names := map[string]string{}
	values := map[string]dynamodbtypes.AttributeValue{}

	placeholder := func(field string) (string, string) {
		nameKey := "#" + field
		valKey := ":" + field
		names[nameKey] = field
		return nameKey, valKey
	}

	if u.Status != nil {
		n, v := placeholder("status")
		sets = append(sets, n+" = "+v)
		values[v] = &dynamodbtypes.AttributeValueMemberS{Value: *u.Status}
	}
	if u.Bib != nil {
		if strings.TrimSpace(*u.Bib) == "" {
			n, _ := placeholder("bib")
			removes = append(removes, n)
		} else {
			n, v := placeholder("bib")
			sets = append(sets, n+" = "+v)
			values[v] = &dynamodbtypes.AttributeValueMemberS{Value: *u.Bib}
		}
	}
	if u.Category != nil {
		n, v := placeholder("category")
		cat, err := attributevalue.MarshalMap(registrationCategory{Gender: u.Category.Gender, AgeGroup: u.Category.AgeGroup})
		if err != nil {
			return nil, fmt.Errorf("marshal category: %w", err)
		}
		sets = append(sets, n+" = "+v)
		values[v] = &dynamodbtypes.AttributeValueMemberM{Value: cat}
	}
	if u.ClearFinish {
		n, _ := placeholder("finishSeconds")
		removes = append(removes, n)
	} else if u.FinishSeconds != nil {
		n, v := placeholder("finishSeconds")
		sets = append(sets, n+" = "+v)
		values[v] = &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", *u.FinishSeconds)}
	}
	if u.ClearSplits {
		n, _ := placeholder("splits")
		removes = append(removes, n)
	} else if u.Splits != nil {
		items := make([]registrationSplitItem, len(u.Splits))
		for i, sp := range u.Splits {
			items[i] = registrationSplitItem{Km: sp.Km, TimeSeconds: sp.TimeSeconds}
		}
		listAV, err := attributevalue.MarshalList(items)
		if err != nil {
			return nil, fmt.Errorf("marshal splits: %w", err)
		}
		n, v := placeholder("splits")
		sets = append(sets, n+" = "+v)
		values[v] = &dynamodbtypes.AttributeValueMemberL{Value: listAV}
	}
	if u.Conditions != nil {
		if strings.TrimSpace(*u.Conditions) == "" {
			n, _ := placeholder("conditions")
			removes = append(removes, n)
		} else {
			n, v := placeholder("conditions")
			sets = append(sets, n+" = "+v)
			values[v] = &dynamodbtypes.AttributeValueMemberS{Value: *u.Conditions}
		}
	}
	if u.Notes != nil {
		if strings.TrimSpace(*u.Notes) == "" {
			n, _ := placeholder("notes")
			removes = append(removes, n)
		} else {
			n, v := placeholder("notes")
			sets = append(sets, n+" = "+v)
			values[v] = &dynamodbtypes.AttributeValueMemberS{Value: *u.Notes}
		}
	}
	if u.ClearPayment {
		n, _ := placeholder("paymentReceivedAt")
		removes = append(removes, n)
	} else if u.PaymentReceivedAt != nil {
		n, v := placeholder("paymentReceivedAt")
		sets = append(sets, n+" = "+v)
		values[v] = &dynamodbtypes.AttributeValueMemberS{Value: u.PaymentReceivedAt.UTC().Format(time.RFC3339)}
	}

	if len(sets) == 0 && len(removes) == 0 {
		return s.GetRegistration(ctx, u.RaceID, u.RunnerID)
	}

	expr := ""
	if len(sets) > 0 {
		expr += "SET " + strings.Join(sets, ", ")
	}
	if len(removes) > 0 {
		if expr != "" {
			expr += " "
		}
		expr += "REMOVE " + strings.Join(removes, ", ")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(s.registrationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"raceId":   &dynamodbtypes.AttributeValueMemberS{Value: u.RaceID},
			"runnerId": &dynamodbtypes.AttributeValueMemberS{Value: u.RunnerID},
		},
		UpdateExpression:         aws.String(expr),
		ExpressionAttributeNames: names,
		ConditionExpression:      aws.String("attribute_exists(raceId) AND attribute_exists(runnerId)"),
		ReturnValues:             dynamodbtypes.ReturnValueAllNew,
	}
	if len(values) > 0 {
		input.ExpressionAttributeValues = values
	}

	out, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		if isConditionalCheckFailed(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update registration: %w", err)
	}
	var item registrationItem
	if err := attributevalue.UnmarshalMap(out.Attributes, &item); err != nil {
		return nil, fmt.Errorf("unmarshal registration: %w", err)
	}
	reg := registrationFromItem(item)
	return &reg, nil
}

func (s *DynamoStore) DeleteRegistration(ctx context.Context, raceID, runnerID string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.registrationsTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"raceId":   &dynamodbtypes.AttributeValueMemberS{Value: raceID},
			"runnerId": &dynamodbtypes.AttributeValueMemberS{Value: runnerID},
		},
	})
	if err != nil {
		return fmt.Errorf("delete registration: %w", err)
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────

func (s *DynamoStore) scanAll(ctx context.Context, table string) ([]map[string]dynamodbtypes.AttributeValue, error) {
	var items []map[string]dynamodbtypes.AttributeValue
	var lastKey map[string]dynamodbtypes.AttributeValue
	for {
		out, err := s.client.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(table),
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, out.Items...)
		if out.LastEvaluatedKey == nil {
			return items, nil
		}
		lastKey = out.LastEvaluatedKey
	}
}

func isConditionalCheckFailed(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "ConditionalCheckFailedException"
}
