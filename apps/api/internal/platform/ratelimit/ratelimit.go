// Package ratelimit is a minimal in-memory, single-process fixed-window rate limiter (PLAN.md
// §10.1.11) — no Redis, no distributed limiter, consistent with this codebase's "self-hosted, no
// extra infra" philosophy (mail, payment). Used only by the external payment API's routes
// (external caller volume is small and bounded — a handful of registered apps, not thousands),
// not a general-purpose rate-limiting layer for the rest of the app.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	count     int
	windowEnd time.Time
}

// Limiter enforces `limit` actions per `window`, per key.
type Limiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*bucket
}

// New builds a Limiter allowing `limit` calls per `window`, per key.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, counters: make(map[string]*bucket)}
}

// Allow reports whether `key` may proceed right now, incrementing its counter if so. Buckets
// past their window reset transparently — no background sweep goroutine (deliberately simple;
// the number of distinct keys — registered app_ids, or caller IPs — stays small in practice).
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.counters[key]
	if !ok || now.After(b.windowEnd) {
		l.counters[key] = &bucket{count: 1, windowEnd: now.Add(l.window)}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}
