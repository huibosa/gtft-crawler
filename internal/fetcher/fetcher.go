package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Fetcher struct {
	client     *http.Client
	userAgent  string
	timeout    time.Duration
	maxRetries int
	verbose    bool
}

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
	Error      error
	Attempts   int
	Duration   time.Duration
}

func NewFetcher(timeout time.Duration, maxRetries, rateLimit int, verbose bool) *Fetcher {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Fetcher{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		timeout:    timeout,
		maxRetries: maxRetries,
		verbose:    verbose,
	}
}

func (f *Fetcher) Fetch(url string) (*FetchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	start := time.Now()
	var lastError error
	var attempts int

	for attempts = 1; attempts <= f.maxRetries; attempts++ {
		if f.verbose {
			fmt.Printf("Fetching attempt %d/%d: %s\n", attempts, f.maxRetries, url)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastError = fmt.Errorf("create request failed: %w", err)
			time.Sleep(f.backoffDuration(attempts))
			continue
		}

		req.Header.Set("User-Agent", f.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
		// Don't set Accept-Encoding - let Go handle decompression automatically
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Cache-Control", "max-age=0")

		resp, err := f.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("HTTP request failed: %w", err)
			time.Sleep(f.backoffDuration(attempts))
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("read response body failed: %w", err)
			time.Sleep(f.backoffDuration(attempts))
			continue
		}

		if resp.StatusCode >= 400 {
			lastError = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			if resp.StatusCode == 404 || resp.StatusCode == 403 {
				// Don't retry on 404 or 403
				break
			}
			time.Sleep(f.backoffDuration(attempts))
			continue
		}

		duration := time.Since(start)

		return &FetchResult{
			URL:        url,
			StatusCode: resp.StatusCode,
			Body:       body,
			Error:      nil,
			Attempts:   attempts,
			Duration:   duration,
		}, nil
	}

	duration := time.Since(start)

	return &FetchResult{
		URL:        url,
		StatusCode: 0,
		Body:       nil,
		Error:      fmt.Errorf("max retries exceeded, last error: %w", lastError),
		Attempts:   attempts - 1,
		Duration:   duration,
	}, nil
}

func (f *Fetcher) backoffDuration(attempt int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s, etc.
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second

	// Cap at 30 seconds
	if backoff > 30*time.Second {
		return 30 * time.Second
	}

	return backoff
}

func (f *Fetcher) SetUserAgent(userAgent string) {
	f.userAgent = userAgent
}

func (f *Fetcher) Close() {
	// The HTTP client doesn't need explicit closing in Go 1.13+
	// But we can use this for cleanup if needed
}
