package provider

import (
	"testing"
	"time"

	"payflow/internal/domain"
)

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	cb := newCircuitBreaker("test", DefaultCircuitBreakerConfig())
	if !cb.Allow() {
		t.Error("expected circuit breaker to allow requests when closed")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 3, ResetTimeout: 1 * time.Second}
	cb := newCircuitBreaker("test", cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.Allow() {
		t.Error("should still allow after 2 failures (threshold is 3)")
	}

	cb.RecordFailure() // 3rd failure — should open
	if cb.Allow() {
		t.Error("should block after 3 failures")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 50 * time.Millisecond}
	cb := newCircuitBreaker("test", cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.Allow() {
		t.Error("should be open")
	}

	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Error("should be half-open after timeout")
	}
	if cb.state != CircuitHalfOpen {
		t.Errorf("expected half-open, got %s", cb.state)
	}
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 50 * time.Millisecond}
	cb := newCircuitBreaker("test", cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow() // transitions to half-open

	cb.RecordSuccess()
	if cb.state != CircuitClosed {
		t.Errorf("expected closed after success in half-open, got %s", cb.state)
	}
	if cb.failures != 0 {
		t.Errorf("expected failures reset to 0, got %d", cb.failures)
	}
}

func TestProviderCircuitBreakers_Registry(t *testing.T) {
	pcb := NewProviderCircuitBreakers(DefaultCircuitBreakerConfig())

	cb1 := pcb.Get(domain.ProviderPaystack)
	cb2 := pcb.Get(domain.ProviderPaystack)

	if cb1 != cb2 {
		t.Error("expected same circuit breaker for same provider")
	}

	cb3 := pcb.Get(domain.ProviderKorapay)
	if cb1 == cb3 {
		t.Error("expected different circuit breakers for different providers")
	}
}
