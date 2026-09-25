package cache

import (
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New(time.Hour)
	c.Set("USD:EUR", map[string]float64{"EUR": 0.9})

	val, ok := c.Get("USD:EUR")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val["EUR"] != 0.9 {
		t.Errorf("expected 0.9, got %v", val["EUR"])
	}
}

func TestGetMiss(t *testing.T) {
	c := New(time.Hour)
	if _, ok := c.Get("missing"); ok {
		t.Error("expected cache miss for unknown key")
	}
}

func TestEntryExpires(t *testing.T) {
	c := New(10 * time.Millisecond)
	c.Set("key", map[string]float64{"X": 1})

	time.Sleep(20 * time.Millisecond)

	if _, ok := c.Get("key"); ok {
		t.Error("expected entry to have expired")
	}
}

func TestSetOverwritesAndResetsTTL(t *testing.T) {
	c := New(time.Hour)
	c.Set("key", map[string]float64{"X": 1})
	c.Set("key", map[string]float64{"X": 2})

	val, ok := c.Get("key")
	if !ok || val["X"] != 2 {
		t.Errorf("expected overwritten value 2, got %v (ok=%v)", val, ok)
	}
}
