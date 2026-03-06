package checks

import (
	"context"
	"net/http"
	"time"
)

// RPCReachable performs a simple HTTP GET to the Solana RPC endpoint.
// Returns true if a response is received within the timeout.
func RPCReachable(rpcURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rpcURL, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	// Solana RPC returns various status codes; any response means it's reachable
	return true
}
