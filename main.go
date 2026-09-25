// Command currency-watcher-backend serves the Currency Watcher REST API:
// GET /api/rates?base=USD&targets=EUR,SGD and GET /api/health.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"currency-watcher-backend/internal/cache"
	"currency-watcher-backend/internal/rates"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Rates are cached for 1 hour so the backend doesn't hit the
	// upstream API on every frontend request.
	rateCache := cache.New(time.Hour)
	client := rates.NewFrankfurterClient(10 * time.Second)
	handler := rates.NewHandler(client, rateCache)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/rates", handler.GetRates)
	mux.HandleFunc("GET /api/health", handler.Health)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           rates.WithCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("currency-watcher backend listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
