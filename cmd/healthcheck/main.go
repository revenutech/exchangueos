// cmd/healthcheck — tiny standalone HTTP healthcheck for distroless images.
//
// The distroless/static base has no shell + no wget/curl, so the docker
// healthcheck cannot use `wget -qO- http://localhost:8084/healthz`.
// This binary fills that gap: it GETs the configured URL and exits 0 only
// on HTTP 200. Used by the docker-compose healthcheck stanza for exchangeos-api.
//
// Usage:
//   healthcheck                       # defaults to http://localhost:8084/healthz
//   healthcheck http://1.2.3.4:8084/readyz
//
// Build cost: ~3 MB statically linked.
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	url := "http://localhost:8084/healthz"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Exit(2)
	}
	os.Exit(0)
}
