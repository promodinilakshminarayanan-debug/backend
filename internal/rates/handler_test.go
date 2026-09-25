package rates

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"currency-watcher-backend/internal/cache"
)

// fakeFetcher lets tests control what the "upstream API" returns
// without making a real network call.
type fakeFetcher struct {
	rates map[string]float64
	err   error
	calls int
}

func (f *fakeFetcher) Fetch(base string, targets []string) (map[string]float64, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.rates, nil
}

func TestGetRates_Success(t *testing.T) {
	fetcher := &fakeFetcher{rates: map[string]float64{"EUR": 0.92}}
	h := NewHandler(fetcher, cache.New(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/rates?base=USD&targets=EUR", nil)
	w := httptest.NewRecorder()

	h.GetRates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body ratesResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Rates["EUR"] != 0.92 {
		t.Errorf("expected EUR rate 0.92, got %v", body.Rates["EUR"])
	}
	if body.Cached {
		t.Error("expected first call not to be marked cached")
	}
}

func TestGetRates_SecondCallHitsCacheNotUpstream(t *testing.T) {
	fetcher := &fakeFetcher{rates: map[string]float64{"EUR": 0.92}}
	h := NewHandler(fetcher, cache.New(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/rates?base=USD&targets=EUR", nil)

	h.GetRates(httptest.NewRecorder(), req)
	h.GetRates(httptest.NewRecorder(), req)

	if fetcher.calls != 1 {
		t.Errorf("expected exactly 1 upstream call due to caching, got %d", fetcher.calls)
	}
}

func TestGetRates_MissingParams(t *testing.T) {
	h := NewHandler(&fakeFetcher{}, cache.New(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/rates", nil)
	w := httptest.NewRecorder()

	h.GetRates(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", w.Code)
	}
}

func TestGetRates_UpstreamError(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("upstream unavailable")}
	h := NewHandler(fetcher, cache.New(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/rates?base=USD&targets=EUR", nil)
	w := httptest.NewRecorder()

	h.GetRates(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 on upstream error, got %d", w.Code)
	}
}

func TestHealth(t *testing.T) {
	h := NewHandler(&fakeFetcher{}, cache.New(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
