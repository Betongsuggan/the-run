package email

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// recordingSender captures the last Message handed to Send, so the test can
// inspect what the renderer would have shipped to SES.
type recordingSender struct {
	last Message
}

func (r *recordingSender) Send(_ context.Context, msg Message) error {
	r.last = msg
	return nil
}

// fakeTemplateStore lets tests pre-populate the store without spinning up
// DynamoDB or LocalStack.
type fakeTemplateStore struct {
	rows map[models.EmailTemplateSlug]*models.EmailTemplate
}

func (f *fakeTemplateStore) GetEmailTemplate(_ context.Context, slug models.EmailTemplateSlug) (*models.EmailTemplate, error) {
	t, ok := f.rows[slug]
	if !ok {
		return nil, nil
	}
	clone := *t
	return &clone, nil
}

func TestRendererFallsBackToSeedWhenStoreEmpty(t *testing.T) {
	store := &fakeTemplateStore{rows: map[models.EmailTemplateSlug]*models.EmailTemplate{}}
	sender := &recordingSender{}
	r := NewRenderer(store, sender)

	err := r.Send(context.Background(), SendOptions{
		Slug:   models.EmailTemplateSlugDSRAccess,
		To:     "user@example.com",
		Locale: LocaleSv,
		Vars:   struct{ Link string }{Link: "https://example.com/?token=abc"},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if sender.last.To != "user@example.com" {
		t.Errorf("To = %q, want user@example.com", sender.last.To)
	}
	if !strings.Contains(sender.last.TextBody, "https://example.com/?token=abc") {
		t.Errorf("body missing substituted link, got: %q", sender.last.TextBody)
	}
	if !strings.Contains(sender.last.Subject, "Hantera") {
		t.Errorf("expected Swedish subject when locale=sv, got %q", sender.last.Subject)
	}
	if sender.last.MessageTag != string(models.EmailTemplateSlugDSRAccess) {
		t.Errorf("MessageTag = %q, want %q", sender.last.MessageTag, models.EmailTemplateSlugDSRAccess)
	}
}

func TestRendererPicksEnglishWhenLocaleEn(t *testing.T) {
	store := &fakeTemplateStore{rows: map[models.EmailTemplateSlug]*models.EmailTemplate{}}
	sender := &recordingSender{}
	r := NewRenderer(store, sender)

	err := r.Send(context.Background(), SendOptions{
		Slug:   models.EmailTemplateSlugDSRAccess,
		To:     "user@example.com",
		Locale: LocaleEn,
		Vars:   struct{ Link string }{Link: "https://example.com/?token=abc"},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !strings.Contains(sender.last.Subject, "Manage") {
		t.Errorf("expected English subject when locale=en, got %q", sender.last.Subject)
	}
}

func TestRendererPrefersPublishedStoreRow(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeTemplateStore{rows: map[models.EmailTemplateSlug]*models.EmailTemplate{
		models.EmailTemplateSlugDSRAccess: {
			Slug:      models.EmailTemplateSlugDSRAccess,
			Status:    models.EmailTemplateStatusPublished,
			Revision:  3,
			SubjectSv: "Custom subject {{.Link}}",
			BodySv:    "Custom body for {{.Link}}",
			SubjectEn: "Custom subject en",
			BodyEn:    "Custom body en",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}
	sender := &recordingSender{}
	r := NewRenderer(store, sender)

	err := r.Send(context.Background(), SendOptions{
		Slug:   models.EmailTemplateSlugDSRAccess,
		To:     "user@example.com",
		Locale: LocaleSv,
		Vars:   struct{ Link string }{Link: "LINK"},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !strings.Contains(sender.last.Subject, "Custom subject LINK") {
		t.Errorf("expected custom subject with substitution, got %q", sender.last.Subject)
	}
	if !strings.Contains(sender.last.TextBody, "Custom body for LINK") {
		t.Errorf("expected custom body with substitution, got %q", sender.last.TextBody)
	}
}

func TestRendererFallsBackFromEnToSvWhenEnEmpty(t *testing.T) {
	store := &fakeTemplateStore{rows: map[models.EmailTemplateSlug]*models.EmailTemplate{
		models.EmailTemplateSlugDSRAccess: {
			Slug:      models.EmailTemplateSlugDSRAccess,
			Status:    models.EmailTemplateStatusPublished,
			Revision:  1,
			SubjectSv: "Svenskt ämne",
			BodySv:    "Svensk text",
			SubjectEn: "",
			BodyEn:    "",
		},
	}}
	sender := &recordingSender{}
	r := NewRenderer(store, sender)

	err := r.Send(context.Background(), SendOptions{
		Slug:   models.EmailTemplateSlugDSRAccess,
		To:     "u@example.com",
		Locale: LocaleEn,
		Vars:   struct{}{},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if sender.last.Subject != "Svenskt ämne" {
		t.Errorf("expected sv fallback when en is empty, got %q", sender.last.Subject)
	}
}

func TestRendererRejectsUnknownSlug(t *testing.T) {
	store := &fakeTemplateStore{rows: map[models.EmailTemplateSlug]*models.EmailTemplate{}}
	r := NewRenderer(store, &recordingSender{})

	err := r.Send(context.Background(), SendOptions{
		Slug:   "no-such-slug",
		To:     "u@example.com",
		Locale: LocaleSv,
		Vars:   struct{}{},
	})
	if err == nil {
		t.Fatal("expected error for unknown slug, got nil")
	}
}
