package rates

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

// cacheStore is the subset of cache.Cache the handler needs. Defined
// locally so tests can supply the real cache without importing it
// into the interface, and so this package doesn't depend on the
// cache package's internals.
type cacheStore interface {
	Get(key string) (map[string]float64, bool)
	Set(key string, value map[string]float64)
}

// Handler serves the currency rate endpoints.
type Handler struct {
	fetcher RateFetcher
	cache   cacheStore
}

// NewHandler wires a rate fetcher and a cache together.
func NewHandler(fetcher RateFetcher, cache cacheStore) *Handler {
	return &Handler{fetcher: fetcher, cache: cache}
}

type ratesResponse struct {
	Base      string             `json:"base"`
	Rates     map[string]float64 `json:"rates"`
	Cached    bool               `json:"cached"`
	Timestamp string             `json:"timestamp"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// cacheKey normalizes base+targets into a stable lookup key regardless
// of the order targets were requested in.
func cacheKey(base string, targets []string) string {
	sorted := append([]string{}, targets...)
	sort.Strings(sorted)
	return base + ":" + strings.Join(sorted, ",")
}

// GetRates handles GET /api/rates?base=USD&targets=EUR,SGD
func (h *Handler) GetRates(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("base")))
	targetsParam := strings.TrimSpace(r.URL.Query().Get("targets"))

	if base == "" || targetsParam == "" {
		writeError(w, http.StatusBadRequest, "base and targets query params are required, e.g. ?base=USD&targets=EUR,SGD")
		return
	}

	targets := splitAndClean(targetsParam)
	if len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "targets must contain at least one currency code")
		return
	}

	log.Printf("incoming rate request: base=%s targets=%v path=%s", base, targets, r.URL.Path)
	key := cacheKey(base, targets)

	if cached, ok := h.cache.Get(key); ok {
		writeJSON(w, http.StatusOK, ratesResponse{
			Base:      base,
			Rates:     cached,
			Cached:    true,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	fresh, err := h.fetcher.Fetch(base, targets)
	if err != nil {
		log.Printf("upstream fetch error: base=%s targets=%v err=%v", base, targets, err)
		writeError(w, http.StatusBadGateway, "could not fetch live rates: "+err.Error())
		return
	}

	h.cache.Set(key, fresh)

	writeJSON(w, http.StatusOK, ratesResponse{
		Base:      base,
		Rates:     fresh,
		Cached:    false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Health handles GET /api/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func splitAndClean(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToUpper(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
