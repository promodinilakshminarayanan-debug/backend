# Currency Watcher — Backend

Go REST API that serves live currency exchange rates to the React frontend,
backed by the free [Frankfurter](https://www.frankfurter.app/) API.

## Prerequisites

- Go 1.24+ (`go version` to check)
- No external Go modules to install — the whole service is built on the
  standard library, so there's nothing to `go get`.

## Run locally

```bash
cd backend
go run .
```

The server starts on `http://localhost:8080` (override with `PORT=9000 go run .`).

## Run the tests

```bash
go test ./...
```

## Endpoints

### `GET /api/health`
```json
{ "status": "ok" }
```

### `GET /api/rates?base=USD&targets=EUR,SGD`
```json
{
  "base": "USD",
  "rates": { "EUR": 0.9241, "SGD": 1.3105 },
  "cached": false,
  "timestamp": "2026-09-25T10:15:00Z"
}
```
`cached: true` means the response came from the in-memory cache instead of
a fresh call to Frankfurter. Missing/empty `base` or `targets` returns
`400`; an upstream failure returns `502`.

## Build a binary

```bash
go build -o server .
./server
```

## Build and run with Docker

```bash
docker build -t currency-watcher-backend .
docker run -p 8080:8080 currency-watcher-backend
```

## Project structure

```
backend/
  main.go                    entry point: wires cache, client, routes, starts the server
  internal/
    cache/
      cache.go                thread-safe in-memory store with TTL expiry
      cache_test.go
    rates/
      client.go                Frankfurter API client (RateFetcher interface)
      handler.go                GET /api/rates and GET /api/health
      handler_test.go           handler tests using a fake fetcher
      cors.go                   CORS middleware for the local frontend
  Dockerfile
  go.mod
```

## Architecture notes

**Caching strategy.** Rates are cached in memory, keyed by `base:sortedTargets`,
for 1 hour. `internal/cache` is a small `sync.RWMutex`-guarded map with
per-entry expiry — reads take a read lock so concurrent requests for the
same or different pairs don't block each other, and an expired entry is
treated as a miss so the next request refetches transparently. This keeps
the service well within Frankfurter's free-tier limits without needing an
external cache like Redis for a project this size.

**Go structure.** `RateFetcher` is a small interface with one implementation
(`FrankfurterClient`) today, which lets `internal/rates`'s handler tests run
against a fake fetcher instead of hitting the network — no test flakiness,
no rate-limit risk from CI runs. Errors from the upstream API are wrapped
with context (`fmt.Errorf("...: %w", err)`) and turned into a `502` with a
JSON error body rather than a bare 500, so the frontend can tell "your
request was malformed" apart from "the rate provider is down."

**No external dependencies.** Routing uses Go 1.22+'s built-in
`http.ServeMux` method patterns (`"GET /api/rates"`), so there was no need
for a third-party router — one less dependency to audit or keep patched.
