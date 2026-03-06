package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClickHouseReachable performs a SELECT 1 query against the ClickHouse HTTP interface.
// Returns true if the query succeeds.
func ClickHouseReachable(clickhouseURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/?query=SELECT+1", clickhouseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}
