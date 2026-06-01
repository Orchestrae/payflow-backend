package provider

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Provider blocked
	CircuitHalfOpen                     // Testing if provider recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Failures before opening circuit (default: 5)
	ResetTimeout     time.Duration // How long to wait before half-open (default: 60s)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     60 * time.Second,
	}
}

// circuitBreaker tracks failure state for a single provider.
type circuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	lastFailure  time.Time
	config       CircuitBreakerConfig
	providerName domain.ProviderName
}

func newCircuitBreaker(name domain.ProviderName, cfg CircuitBreakerConfig) *circuitBreaker {
	return &circuitBreaker{
		state:        CircuitClosed,
		config:       cfg,
		providerName: name,
	}
}

// Allow returns true if the circuit allows a request through.
func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = CircuitHalfOpen
			log.Info().Str("provider", string(cb.providerName)).Msg("Circuit breaker half-open — testing provider")
			return true
		}
		return false
	case CircuitHalfOpen:
		return true // Allow one request through to test
	}
	return false
}

// RecordSuccess records a successful operation.
func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		log.Info().Str("provider", string(cb.providerName)).Msg("Circuit breaker closed — provider recovered")
	}
	cb.state = CircuitClosed
	cb.failures = 0
}

// RecordFailure records a failed operation.
func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.config.FailureThreshold {
		cb.state = CircuitOpen
		log.Warn().
			Str("provider", string(cb.providerName)).
			Int("failures", cb.failures).
			Dur("reset_timeout", cb.config.ResetTimeout).
			Msg("Circuit breaker OPEN — provider blocked")
	}
}

// ProviderCircuitBreakers manages circuit breakers for all providers.
type ProviderCircuitBreakers struct {
	breakers map[domain.ProviderName]*circuitBreaker
	mu       sync.RWMutex
	config   CircuitBreakerConfig
}

// NewProviderCircuitBreakers creates a new circuit breaker registry.
func NewProviderCircuitBreakers(cfg CircuitBreakerConfig) *ProviderCircuitBreakers {
	return &ProviderCircuitBreakers{
		breakers: make(map[domain.ProviderName]*circuitBreaker),
		config:   cfg,
	}
}

// Get returns the circuit breaker for a provider (creates if needed).
func (pcb *ProviderCircuitBreakers) Get(name domain.ProviderName) *circuitBreaker {
	pcb.mu.RLock()
	cb, exists := pcb.breakers[name]
	pcb.mu.RUnlock()

	if exists {
		return cb
	}

	pcb.mu.Lock()
	defer pcb.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := pcb.breakers[name]; exists {
		return cb
	}

	cb = newCircuitBreaker(name, pcb.config)
	pcb.breakers[name] = cb
	return cb
}

// IsAvailable checks if a provider's circuit is not open.
func (pcb *ProviderCircuitBreakers) IsAvailable(name domain.ProviderName) bool {
	return pcb.Get(name).Allow()
}

// ErrCircuitOpen is returned when a provider's circuit breaker is open.
var ErrCircuitOpen = fmt.Errorf("circuit breaker open: provider temporarily unavailable")
