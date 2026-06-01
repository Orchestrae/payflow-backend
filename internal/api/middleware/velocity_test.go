package middleware

import (
	"testing"
	"time"
)

func TestTransferCounter_Cleanup(t *testing.T) {
	now := time.Now()
	c := &transferCounter{
		hourly: []time.Time{
			now.Add(-2 * time.Hour), // expired
			now.Add(-30 * time.Minute), // valid
			now.Add(-5 * time.Minute),  // valid
		},
		daily: []time.Time{
			now.Add(-25 * time.Hour), // expired
			now.Add(-12 * time.Hour), // valid
			now.Add(-1 * time.Hour),  // valid
		},
	}

	c.cleanup(now)

	if len(c.hourly) != 2 {
		t.Errorf("expected 2 hourly entries after cleanup, got %d", len(c.hourly))
	}
	if len(c.daily) != 2 {
		t.Errorf("expected 2 daily entries after cleanup, got %d", len(c.daily))
	}
}

func TestTransferCounter_AllExpired(t *testing.T) {
	now := time.Now()
	c := &transferCounter{
		hourly: []time.Time{
			now.Add(-3 * time.Hour),
			now.Add(-2 * time.Hour),
		},
		daily: []time.Time{
			now.Add(-48 * time.Hour),
		},
	}

	c.cleanup(now)

	if len(c.hourly) != 0 {
		t.Errorf("expected 0 hourly entries, got %d", len(c.hourly))
	}
	if len(c.daily) != 0 {
		t.Errorf("expected 0 daily entries, got %d", len(c.daily))
	}
}
