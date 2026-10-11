package ajp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPPageLoader fetches sourceURL with a plain GET — a real User-Agent
// (never a browser's), a client-side timeout, and exactly one request per
// call. If this importer ever needs more than one request in a single
// run, every later one must wait out robots.txt's own Crawl-delay: 10
// before firing — there is deliberately no retry loop here for that same
// reason.
type HTTPPageLoader struct {
	client *http.Client
}

func NewHTTPPageLoader(timeout time.Duration) *HTTPPageLoader {
	return &HTTPPageLoader{client: &http.Client{Timeout: timeout}}
}

func (l *HTTPPageLoader) Load(ctx context.Context) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	return string(body), resp.StatusCode, nil
}
