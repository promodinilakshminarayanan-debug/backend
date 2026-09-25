package rates

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RateFetcher fetches live rates from an upstream provider. It's an
// interface so handlers can be tested against a fake instead of the
// real network call.
type RateFetcher interface {
	Fetch(base string, targets []string) (map[string]float64, error)
}

// FrankfurterClient calls the free https://www.frankfurter.app API.
type FrankfurterClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewFrankfurterClient builds a client with the given request timeout.
func NewFrankfurterClient(timeout time.Duration) *FrankfurterClient {
	return &FrankfurterClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    "https://api.frankfurter.app",
	}
}

type frankfurterResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

// Fetch calls GET /latest?from=<base>&to=<targets> and returns the
// target currency -> rate map.
func (f *FrankfurterClient) Fetch(base string, targets []string) (map[string]float64, error) {
	q := url.Values{}
	q.Set("from", base)
	q.Set("to", strings.Join(targets, ","))

	reqURL := fmt.Sprintf("%s/latest?%s", f.baseURL, q.Encode())
	log.Printf("requesting live rates: base=%s targets=%v url=%s", base, targets, reqURL)

	resp, err := f.httpClient.Get(reqURL)
	if err != nil {
		log.Printf("frankfurter request failed: base=%s targets=%v err=%v", base, targets, err)
		return nil, fmt.Errorf("calling frankfurter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("frankfurter status=%d but failed to read response body: %v", resp.StatusCode, readErr)
		} else {
			log.Printf("frankfurter returned status=%d for base=%s targets=%v body=%s", resp.StatusCode, base, targets, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("frankfurter returned status %d", resp.StatusCode)
	}

	var parsed frankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		log.Printf("failed to decode frankfurter response for base=%s targets=%v: %v", base, targets, err)
		return nil, fmt.Errorf("decoding frankfurter response: %w", err)
	}

	log.Printf("live rates fetched successfully: base=%s targets=%v rates=%v", base, targets, parsed.Rates)
	return parsed.Rates, nil
}
