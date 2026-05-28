package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// CacheService provides cache-aside functionality.
// If redis is nil, all operations are no-ops (graceful degradation).
type CacheService struct {
	redis *RedisClient
}

// NewCacheService creates a cache service. Pass nil for no-cache mode.
func NewCacheService(redis *RedisClient) *CacheService {
	return &CacheService{redis: redis}
}

// GetOrLoad implements cache-aside: try cache first, fall back to loader, store result.
// The loader function should return the data from the database.
// T must be JSON-serializable.
func GetOrLoad[T any](cs *CacheService, ctx context.Context, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var zero T

	// No cache available — go straight to DB
	if cs == nil || cs.redis == nil {
		return loader()
	}

	// Try cache
	cached, err := cs.redis.Get(ctx, key)
	if err == nil {
		var result T
		if json.Unmarshal([]byte(cached), &result) == nil {
			return result, nil
		}
	} else if err != redis.Nil {
		log.Warn().Err(err).Str("key", key).Msg("cache get error")
	}

	// Cache miss — load from DB
	result, err := loader()
	if err != nil {
		return zero, err
	}

	// Store in cache (best-effort, don't fail the request)
	if data, marshalErr := json.Marshal(result); marshalErr == nil {
		if setErr := cs.redis.Set(ctx, key, string(data), ttl); setErr != nil {
			log.Warn().Err(setErr).Str("key", key).Msg("cache set error")
		}
	}

	return result, nil
}

// Invalidate removes a cache entry.
func (cs *CacheService) Invalidate(ctx context.Context, key string) {
	if cs == nil || cs.redis == nil {
		return
	}
	if err := cs.redis.Delete(ctx, key); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("cache invalidate error")
	}
}

// InvalidatePattern removes all cache entries matching a pattern.
func (cs *CacheService) InvalidatePattern(ctx context.Context, pattern string) {
	if cs == nil || cs.redis == nil {
		return
	}
	if err := cs.redis.DeletePattern(ctx, pattern); err != nil {
		log.Warn().Err(err).Str("pattern", pattern).Msg("cache invalidate pattern error")
	}
}
