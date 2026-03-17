package altcache

import "sync/atomic"

// Metrics tracks cache performance across all layers.
type Metrics struct {
	L1Hits int64 // sync.Map hits (atomic)
	L2Hits int64 // Redis hits (atomic)
	Misses int64 // No hit in any layer (atomic)
	Writes int64 // Put operations (atomic)
}

// MetricsSnapshot is the serializable cache performance snapshot.
type MetricsSnapshot struct {
	L1Hits  int64   `json:"l1_hits"`
	L2Hits  int64   `json:"l2_hits"`
	Misses  int64   `json:"misses"`
	Writes  int64   `json:"writes"`
	HitRate float64 `json:"hit_rate_pct"`
}

// Snapshot returns a point-in-time view of cache metrics.
func (c *Cache) Snapshot() MetricsSnapshot {
	l1 := atomic.LoadInt64(&c.Metrics.L1Hits)
	l2 := atomic.LoadInt64(&c.Metrics.L2Hits)
	miss := atomic.LoadInt64(&c.Metrics.Misses)
	writes := atomic.LoadInt64(&c.Metrics.Writes)

	totalAttempts := l1 + l2 + miss
	hitRate := 0.0
	if totalAttempts > 0 {
		hitRate = float64(l1+l2) / float64(totalAttempts) * 100.0
	}

	return MetricsSnapshot{
		L1Hits:  l1,
		L2Hits:  l2,
		Misses:  miss,
		Writes:  writes,
		HitRate: hitRate,
	}
}
