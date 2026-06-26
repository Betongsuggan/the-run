package ratelimit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/ratelimit"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// memStore is an in-memory store stub that implements just the rate-limit
// surface ratelimit.{Check,Record} touch. Embeds store.Store so unrelated
// methods panic loudly if ever invoked.
type memStore struct {
	store.Store
	hits map[string][]hit
}

type hit struct {
	at      time.Time
	expires time.Time
}

func newMemStore() *memStore {
	return &memStore{hits: map[string][]hit{}}
}

func (m *memStore) RecordRateLimitHit(_ context.Context, bucket string, at time.Time, ttl time.Duration) error {
	m.hits[bucket] = append(m.hits[bucket], hit{at: at, expires: at.Add(ttl)})
	return nil
}

func (m *memStore) CountActiveRateLimitHits(_ context.Context, bucket string, now time.Time) (int, error) {
	n := 0
	for _, h := range m.hits[bucket] {
		if h.expires.After(now) {
			n++
		}
	}
	return n, nil
}

func TestCheckPassesUnderLimit(t *testing.T) {
	t.Parallel()
	s := newMemStore()
	ctx := context.Background()
	now := time.Now().UTC()
	for i := range 4 {
		if err := ratelimit.Record(ctx, s, "register#1.2.3.4", now, time.Minute); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	if err := ratelimit.Check(ctx, s, "register#1.2.3.4", 5, now); err != nil {
		t.Fatalf("4 hits under max=5 should pass, got %v", err)
	}
}

func TestCheckRejectsAtAndAboveLimit(t *testing.T) {
	t.Parallel()
	s := newMemStore()
	ctx := context.Background()
	now := time.Now().UTC()
	for i := range 5 {
		if err := ratelimit.Record(ctx, s, "register#1.2.3.4", now, time.Minute); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	if err := ratelimit.Check(ctx, s, "register#1.2.3.4", 5, now); !errors.Is(err, ratelimit.ErrRateLimited) {
		t.Fatalf("5 hits at max=5 should reject, got %v", err)
	}
}

func TestCheckRecoversAfterWindow(t *testing.T) {
	t.Parallel()
	s := newMemStore()
	ctx := context.Background()
	t0 := time.Now().UTC()
	for i := range 5 {
		if err := ratelimit.Record(ctx, s, "register#1.2.3.4", t0, time.Minute); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	// One minute and one second later — all five hits have aged out.
	later := t0.Add(time.Minute + time.Second)
	if err := ratelimit.Check(ctx, s, "register#1.2.3.4", 5, later); err != nil {
		t.Fatalf("expected window expiry to drop the count to 0, got %v", err)
	}
}

func TestCheckIsolatesBuckets(t *testing.T) {
	t.Parallel()
	s := newMemStore()
	ctx := context.Background()
	now := time.Now().UTC()
	for i := range 5 {
		if err := ratelimit.Record(ctx, s, "register#1.2.3.4", now, time.Minute); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	if err := ratelimit.Check(ctx, s, "register#9.9.9.9", 5, now); err != nil {
		t.Fatalf("different IP should not be limited, got %v", err)
	}
}
