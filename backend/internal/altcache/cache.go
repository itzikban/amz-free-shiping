package altcache

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"free-ship-checker-go/internal/checker"
)

// Cache implements a three-layer cache for alternative products:
// L1: sync.Map (in-process, 1h TTL)
// L2: Redis (cross-instance, 24h TTL)
// L3: PostgreSQL (persistent, 24h TTL + future RAG)
type Cache struct {
	l1       sync.Map      // key: string(asin) → *l1Entry
	l1TTL    time.Duration // default: 1h
	redis    *redis.Client // nil = skip L2
	redisTTL time.Duration // default: 24h
	db       *pgxpool.Pool // nil = skip DB write
	dbTTL    time.Duration // default: 24h
	Metrics  *Metrics      // atomic counters
}

// l1Entry holds a cached result with expiration time.
type l1Entry struct {
	alts      []checker.Alternative
	expiresAt time.Time
}

// New creates a new Cache instance. Either rdb or db (or both) can be nil.
func New(rdb *redis.Client, db *pgxpool.Pool) *Cache {
	return &Cache{
		l1:       sync.Map{},
		l1TTL:    1 * time.Hour,
		redis:    rdb,
		redisTTL: 24 * time.Hour,
		db:       db,
		dbTTL:    24 * time.Hour,
		Metrics:  &Metrics{},
	}
}

// Get looks up alternatives by ASIN across all layers: L1 → L2 → L3 → miss.
// Returns (alternatives, found bool). Never errors; degrades gracefully if layers unavailable.
func (c *Cache) Get(ctx context.Context, asin string) ([]checker.Alternative, bool) {
	if c == nil || asin == "" {
		return nil, false
	}

	// L1: Check sync.Map
	if v, ok := c.l1.Load(asin); ok {
		entry, typed := v.(*l1Entry)
		if !typed {
			c.l1.Delete(asin)
		} else if time.Now().Before(entry.expiresAt) {
			atomic.AddInt64(&c.Metrics.L1Hits, 1)
			return entry.alts, true
		} else {
			// Expired, delete from L1
			c.l1.Delete(asin)
		}
	}

	// L2: Check Redis
	if c.redis != nil {
		redisKey := "altcache:" + asin
		val, err := c.redis.Get(ctx, redisKey).Result()
		if err == nil {
			var alts []checker.Alternative
			if err := json.Unmarshal([]byte(val), &alts); err == nil {
				// Populate L1 from L2
				c.l1.Store(asin, &l1Entry{
					alts:      alts,
					expiresAt: time.Now().Add(c.l1TTL),
				})
				atomic.AddInt64(&c.Metrics.L2Hits, 1)
				return alts, true
			}
		}
		// redis.Nil is a miss, other errors are silent
	}

	// L3: Check PostgreSQL
	if c.db != nil {
		var raw string
		err := c.db.QueryRow(ctx, `
			SELECT results_json
			FROM alternative_searches
			WHERE source_asin = $1
			  AND expires_at > NOW()
		`, asin).Scan(&raw)
		if err == nil {
			var alts []checker.Alternative
			if uerr := json.Unmarshal([]byte(raw), &alts); uerr == nil {
				c.l1.Store(asin, &l1Entry{
					alts:      alts,
					expiresAt: time.Now().Add(c.l1TTL),
				})
				atomic.AddInt64(&c.Metrics.L3Hits, 1)
				return alts, true
			}
		} else if err != sql.ErrNoRows {
			log.Printf("[altcache] db read failed for %s: %v", asin, err)
		}
	}

	// Miss
	atomic.AddInt64(&c.Metrics.Misses, 1)
	return nil, false
}

// Put stores alternatives in all cache layers (L1 synchronously, L2+L3 async).
func (c *Cache) Put(ctx context.Context, asin, query string, alts []checker.Alternative) {
	if c == nil || asin == "" {
		return
	}
	if alts == nil {
		alts = []checker.Alternative{}
	}

	// L1: synchronous
	c.l1.Store(asin, &l1Entry{
		alts:      alts,
		expiresAt: time.Now().Add(c.l1TTL),
	})
	atomic.AddInt64(&c.Metrics.Writes, 1)

	// L2 + L3: fire and forget (use detached context)
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Redis SET
		if c.redis != nil {
			redisKey := "altcache:" + asin
			b, _ := json.Marshal(alts)
			if err := c.redis.Set(bg, redisKey, string(b), c.redisTTL).Err(); err != nil {
				log.Printf("[altcache] redis set failed for %s: %v", asin, err)
			}
		}

		// DB upsert
		if c.db != nil {
			b, _ := json.Marshal(alts)
			ttlSeconds := int(c.dbTTL.Seconds())
			if _, err := c.db.Exec(bg, `
				INSERT INTO alternative_searches
				  (source_asin, search_query, results_json, result_count, expires_at)
				VALUES ($1, $2, $3, $4, NOW() + INTERVAL '1 second' * $5)
				ON CONFLICT (source_asin) DO UPDATE SET
				  search_query  = EXCLUDED.search_query,
				  results_json  = EXCLUDED.results_json,
				  result_count  = EXCLUDED.result_count,
				  expires_at    = EXCLUDED.expires_at
			`, asin, query, string(b), len(alts), ttlSeconds); err != nil {
				log.Printf("[altcache] db upsert failed for %s: %v", asin, err)
			}
		}
	}()
}
