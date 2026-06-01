package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// TransferVelocityConfig defines transfer rate limits per business.
type TransferVelocityConfig struct {
	MaxPerHour int // Maximum transfers per hour per business
	MaxPerDay  int // Maximum transfers per day per business
}

type transferCounter struct {
	hourly    []time.Time
	daily     []time.Time
	mu        sync.Mutex
}

func (c *transferCounter) cleanup(now time.Time) {
	hourAgo := now.Add(-time.Hour)
	dayAgo := now.Add(-24 * time.Hour)

	// Trim hourly
	newHourly := c.hourly[:0]
	for _, t := range c.hourly {
		if t.After(hourAgo) {
			newHourly = append(newHourly, t)
		}
	}
	c.hourly = newHourly

	// Trim daily
	newDaily := c.daily[:0]
	for _, t := range c.daily {
		if t.After(dayAgo) {
			newDaily = append(newDaily, t)
		}
	}
	c.daily = newDaily
}

// TransferVelocityLimiter limits the number of transfers per business per time window.
func TransferVelocityLimiter(cfg TransferVelocityConfig) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		counters = make(map[uint]*transferCounter)
	)

	// Periodic cleanup
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for _, c := range counters {
				c.mu.Lock()
				c.cleanup(now)
				c.mu.Unlock()
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			businessID := claims.BusinessID
			now := time.Now()

			mu.Lock()
			counter, exists := counters[businessID]
			if !exists {
				counter = &transferCounter{}
				counters[businessID] = counter
			}
			mu.Unlock()

			counter.mu.Lock()
			counter.cleanup(now)

			// Check hourly limit
			if cfg.MaxPerHour > 0 && len(counter.hourly) >= cfg.MaxPerHour {
				counter.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "transfer velocity limit exceeded",
					"limit":         cfg.MaxPerHour,
					"window":        "1 hour",
					"retry_after_s": 3600,
				})
				return
			}

			// Check daily limit
			if cfg.MaxPerDay > 0 && len(counter.daily) >= cfg.MaxPerDay {
				counter.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "86400")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "daily transfer limit exceeded",
					"limit":         cfg.MaxPerDay,
					"window":        "24 hours",
					"retry_after_s": 86400,
				})
				return
			}

			// Record this transfer
			counter.hourly = append(counter.hourly, now)
			counter.daily = append(counter.daily, now)
			counter.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
