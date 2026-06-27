package email

import (
	"context"
	"log"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// SeedStore is the subset of store.Store the seed routine needs. Declared
// here so email doesn't import store directly.
type SeedStore interface {
	GetEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug) (*models.EmailTemplate, error)
	CreateEmailTemplate(ctx context.Context, t models.EmailTemplate) error
}

// SeedTemplates writes a published row for every slug in
// models.KnownEmailTemplateSlugs that doesn't already exist in the store.
// Idempotent — re-running on a populated DB is a no-op. Called once at
// startup from cmd/api/main.go so a fresh deploy ships with usable copy.
//
// Errors per slug are logged and swallowed; we don't want a single failing
// upsert to crash the API on startup. Worst case the renderer falls back to
// SeededTemplate at send time.
func SeedTemplates(ctx context.Context, s SeedStore) {
	now := time.Now().UTC()
	for _, seed := range AllSeededTemplates() {
		existing, err := s.GetEmailTemplate(ctx, seed.Slug)
		if err != nil {
			log.Printf("email seed: lookup failed slug=%s err=%v", seed.Slug, err)
			continue
		}
		if existing != nil {
			continue
		}
		t := models.EmailTemplate{
			Slug:               seed.Slug,
			DisplayName:        seed.DisplayName,
			Status:             models.EmailTemplateStatusPublished,
			Revision:           1,
			SubjectSv:          seed.SubjectSv,
			BodySv:             seed.BodySv,
			SubjectEn:          seed.SubjectEn,
			BodyEn:             seed.BodyEn,
			AvailableVariables: seed.AvailableVariables,
			CreatedAt:          now,
			UpdatedAt:          now,
			PublishedAt:        &now,
			PublishedBy:        "seed",
			LastEditedBy:       "seed",
		}
		if err := s.CreateEmailTemplate(ctx, t); err != nil {
			// Concurrent boots can race; whichever won keeps its row. Log
			// and move on — we don't want a single bad slug to crash the
			// API.
			log.Printf("email seed: create slug=%s err=%v", seed.Slug, err)
			continue
		}
		log.Printf("email seed: published default body slug=%s", seed.Slug)
	}
}
