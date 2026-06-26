// Package ratelimit implements a per-bucket sliding-window throttle backed
// by the rate-limit DynamoDB table. It's the AWS-native, zero-extra-cost
// alternative to AWS WAF rate rules — see GDPR plan A0.7.
//
// Callers compose a bucket key (e.g. "register#<ip>") so the same table can
// host multiple independent throttles. The window is the same as the TTL
// applied to each recorded hit, so the "active" count auto-decays without
// any sweep job.
package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// ErrRateLimited is returned by Check when the bucket has reached its cap.
// Callers should map to HTTP 429.
var ErrRateLimited = errors.New("rate limit exceeded")

// Check refuses the action when the bucket has already accumulated `max`
// or more hits inside the current window. Call BEFORE doing any expensive
// or side-effect-producing work. Idempotent — does not write.
func Check(ctx context.Context, s store.Store, bucket string, max int, now time.Time) error {
	count, err := s.CountActiveRateLimitHits(ctx, bucket, now)
	if err != nil {
		return err
	}
	if count >= max {
		return ErrRateLimited
	}
	return nil
}

// Record stamps one hit for the bucket, expiring after `window`. The hit
// counts toward future Check calls until it falls out of the window via
// DynamoDB TTL (or the bridging FilterExpression in
// CountActiveRateLimitHits, whichever comes first).
func Record(ctx context.Context, s store.Store, bucket string, at time.Time, window time.Duration) error {
	return s.RecordRateLimitHit(ctx, bucket, at, window)
}
