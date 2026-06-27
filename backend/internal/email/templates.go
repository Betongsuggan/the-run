package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// LocaleSv and LocaleEn are the two locales the renderer supports. Recipients
// without a Locale field default to LocaleSv (matches the brand language).
const (
	LocaleSv = "sv"
	LocaleEn = "en"
)

// normalizeLocale falls back to Swedish on any unknown value. Keeps a typo in
// account.Locale from silently swallowing the email.
func normalizeLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case LocaleEn:
		return LocaleEn
	default:
		return LocaleSv
	}
}

// MessageTagFor returns the SES MessageTag for a given template slug. Reused
// by the existing CloudWatch dashboards: each tag is a CloudWatch dimension.
func MessageTagFor(slug models.EmailTemplateSlug) string {
	return string(slug)
}

// templateStore is the subset of store.Store the renderer needs. Declared
// here (not imported) so the email package stays cycle-free with store.
type templateStore interface {
	GetEmailTemplate(ctx context.Context, slug models.EmailTemplateSlug) (*models.EmailTemplate, error)
}

// Renderer composes and sends a transactional email. Call sites hand it the
// slug, the recipient's locale, the recipient address, and a variables struct
// — it loads the published template body from the store, substitutes via
// text/template, and sends via the wrapped Sender.
//
// Falls back to the seeded body (see seedBody) when the store has no
// published row yet. This keeps the first send after a fresh deploy working
// even before the seed migration runs.
type Renderer struct {
	store  templateStore
	sender Sender

	mu    sync.RWMutex
	cache map[cacheKey]*compiledTemplate
}

type cacheKey struct {
	Slug     models.EmailTemplateSlug
	Locale   string
	Revision int
}

type compiledTemplate struct {
	Subject *template.Template
	Body    *template.Template
}

// NewRenderer wires a renderer around the given store and underlying Sender.
// The underlying sender can be SES, NoOp, or the audit-wrapped variant — the
// renderer doesn't care.
func NewRenderer(s templateStore, sender Sender) *Renderer {
	return &Renderer{
		store:  s,
		sender: sender,
		cache:  map[cacheKey]*compiledTemplate{},
	}
}

// SendOptions describes one send. To is the recipient; Locale is "sv"/"en"
// from account.Locale (empty defaults to "sv"); Vars is whatever struct the
// template references (e.g. struct{ Link, RunnerName string }).
type SendOptions struct {
	Slug   models.EmailTemplateSlug
	To     string
	Locale string
	Vars   any
}

// Send composes and dispatches the templated email. Errors propagate from
// either the store lookup or the underlying Sender.
func (r *Renderer) Send(ctx context.Context, opts SendOptions) error {
	if opts.To == "" {
		return errors.New("email renderer: To is required")
	}
	if !models.IsKnownEmailTemplateSlug(string(opts.Slug)) {
		return fmt.Errorf("email renderer: unknown slug %q", opts.Slug)
	}
	locale := normalizeLocale(opts.Locale)

	subject, body, err := r.resolve(ctx, opts.Slug, locale)
	if err != nil {
		return err
	}

	subjectStr, err := executeTemplate(subject, opts.Vars)
	if err != nil {
		return fmt.Errorf("render subject for %s: %w", opts.Slug, err)
	}
	bodyStr, err := executeTemplate(body, opts.Vars)
	if err != nil {
		return fmt.Errorf("render body for %s: %w", opts.Slug, err)
	}
	return r.sender.Send(ctx, Message{
		To:         opts.To,
		Subject:    subjectStr,
		TextBody:   bodyStr,
		MessageTag: MessageTagFor(opts.Slug),
	})
}

// resolve returns the compiled subject + body for (slug, locale). Uses the
// published row from the store; falls back to the seeded body when nothing is
// published yet. Caches compiled templates per (slug, locale, revision) so a
// hot loop doesn't pay parse cost every send.
func (r *Renderer) resolve(ctx context.Context, slug models.EmailTemplateSlug, locale string) (*template.Template, *template.Template, error) {
	tmpl, err := r.store.GetEmailTemplate(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("load email template %s: %w", slug, err)
	}

	if tmpl != nil && tmpl.Status == models.EmailTemplateStatusPublished {
		key := cacheKey{Slug: slug, Locale: locale, Revision: tmpl.Revision}
		r.mu.RLock()
		hit, ok := r.cache[key]
		r.mu.RUnlock()
		if ok {
			return hit.Subject, hit.Body, nil
		}
		subject, body := pickLocale(tmpl, locale)
		st, err := template.New(string(slug) + ":subject").Option("missingkey=zero").Parse(subject)
		if err != nil {
			return nil, nil, fmt.Errorf("parse subject for %s: %w", slug, err)
		}
		bt, err := template.New(string(slug) + ":body").Option("missingkey=zero").Parse(body)
		if err != nil {
			return nil, nil, fmt.Errorf("parse body for %s: %w", slug, err)
		}
		r.mu.Lock()
		r.cache[key] = &compiledTemplate{Subject: st, Body: bt}
		r.mu.Unlock()
		return st, bt, nil
	}

	// No published row — use the in-code seed so the first deploy still
	// sends real emails. Not cached: this code path should normally only
	// hit until the seed routine populates DynamoDB.
	seed, ok := SeededTemplate(slug)
	if !ok {
		return nil, nil, fmt.Errorf("no published template for %s and no seed available", slug)
	}
	subject, body := pickLocaleStrings(seed.SubjectSv, seed.BodySv, seed.SubjectEn, seed.BodyEn, locale)
	st, err := template.New(string(slug) + ":subject").Option("missingkey=zero").Parse(subject)
	if err != nil {
		return nil, nil, fmt.Errorf("parse seed subject for %s: %w", slug, err)
	}
	bt, err := template.New(string(slug) + ":body").Option("missingkey=zero").Parse(body)
	if err != nil {
		return nil, nil, fmt.Errorf("parse seed body for %s: %w", slug, err)
	}
	return st, bt, nil
}

// pickLocale returns (subject, body) for the requested locale, with an empty-
// string fallback to the other locale. Empty `en` fields fall back to `sv` so
// admins can publish a template with only Swedish copy and still deliver.
func pickLocale(t *models.EmailTemplate, locale string) (string, string) {
	return pickLocaleStrings(t.SubjectSv, t.BodySv, t.SubjectEn, t.BodyEn, locale)
}

func pickLocaleStrings(subjectSv, bodySv, subjectEn, bodyEn, locale string) (string, string) {
	if locale == LocaleEn {
		subject := subjectEn
		if subject == "" {
			subject = subjectSv
		}
		body := bodyEn
		if body == "" {
			body = bodySv
		}
		return subject, body
	}
	return subjectSv, bodySv
}

func executeTemplate(t *template.Template, vars any) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
