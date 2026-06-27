// Command seed-policy bootstraps a privacy-policy draft directly in DynamoDB
// without going through the admin API. The intended use is the first deploy
// (or a fresh LocalStack table): an admin would otherwise have to paste the
// full markdown into the admin GUI to get a policy in place; this lets us
// keep the policy text in git and load it with one command.
//
//	just seed-policy
//
// The result is always a Draft — review and publish via /admin/policies so
// the publish-flow audit row is captured properly. Idempotent on the slug:
// rerunning with the same slug is a no-op unless -force is passed.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

func main() {
	slug := flag.String("slug", "", "Policy slug, e.g. 2026-06-26 (required)")
	kind := flag.String("kind", string(models.DefaultPolicyKind), "Policy kind, e.g. privacy (defaults to privacy)")
	svFile := flag.String("sv", "", "Path to Swedish body markdown file (required)")
	enFile := flag.String("en", "", "Path to English body markdown file (required)")
	effective := flag.String("effective-from", "", "RFC3339 timestamp; defaults to now")
	force := flag.Bool("force", false, "Overwrite the body of an existing same-(kind,slug) draft (refuses to touch published/archived rows)")
	flag.Parse()

	if *slug == "" || *svFile == "" || *enFile == "" {
		fmt.Fprintln(os.Stderr, "usage: seed-policy -slug=SLUG -sv=PATH -en=PATH [-effective-from=RFC3339] [-force]")
		os.Exit(2)
	}

	svBody, err := readMarkdown(*svFile)
	if err != nil {
		log.Fatalf("read sv body: %v", err)
	}
	enBody, err := readMarkdown(*enFile)
	if err != nil {
		log.Fatalf("read en body: %v", err)
	}

	var effFrom time.Time
	if *effective == "" {
		effFrom = time.Now().UTC()
	} else {
		t, err := time.Parse(time.RFC3339, *effective)
		if err != nil {
			log.Fatalf("parse -effective-from: %v", err)
		}
		effFrom = t.UTC()
	}

	ctx := context.Background()
	s, err := store.NewDynamoStoreFromEnv(ctx)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	policyKind := models.PolicyKind(strings.TrimSpace(*kind))
	if policyKind == "" {
		policyKind = models.DefaultPolicyKind
	}

	existing, err := s.GetPolicyBySlug(ctx, policyKind, *slug)
	if err != nil {
		log.Fatalf("lookup slug: %v", err)
	}
	if existing != nil {
		if !*force {
			fmt.Printf("policy %s/%q already exists (id=%s status=%s revision=%d). Rerun with -force to overwrite its draft body.\n",
				policyKind, *slug, existing.ID, existing.Status, existing.Revision)
			return
		}
		if existing.Status != models.PolicyStatusDraft {
			log.Fatalf("policy %s/%q is %s; refusing to overwrite. Create a new version via the admin GUI instead.",
				policyKind, *slug, existing.Status)
		}
		editor := "seed-policy-cli"
		if err := s.UpdatePolicyDraft(ctx, existing.ID, svBody, enBody, editor); err != nil {
			log.Fatalf("update draft body: %v", err)
		}
		fmt.Printf("updated existing draft %s/%q (id=%s). Open /admin/policies to review and publish.\n", policyKind, *slug, existing.ID)
		return
	}

	now := time.Now().UTC()
	p := models.Policy{
		ID:            uuid.NewString(),
		Kind:          policyKind,
		Slug:          *slug,
		Status:        models.PolicyStatusDraft,
		Revision:      1,
		EffectiveFrom: effFrom,
		BodySv:        svBody,
		BodyEn:        enBody,
		CreatedAt:     now,
		UpdatedAt:     now,
		LastEditedBy:  "seed-policy-cli",
	}
	if err := s.CreatePolicyDraft(ctx, p); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			log.Fatalf("policy %s/%q taken by a concurrent writer", policyKind, *slug)
		}
		log.Fatalf("create policy draft: %v", err)
	}
	fmt.Printf("created draft policy:\n  id=%s\n  kind=%s\n  slug=%s\n  revision=%d\n  effective-from=%s\nOpen /admin/policies to review and publish.\n",
		p.ID, p.Kind, p.Slug, p.Revision, p.EffectiveFrom.Format(time.RFC3339))
}

func readMarkdown(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return string(body), nil
}
