package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	versionpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/version"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type baseClient struct {
	baseURL    *url.URL
	httpClient httpDoer
	retry      RetryOptions
}

type Option func(*baseClient)

// WithHTTPClient overrides the default HTTP client used for requests.
func WithHTTPClient(h httpDoer) Option {
	return func(b *baseClient) {
		if h != nil {
			b.httpClient = h
		}
	}
}

func newBase(endpoint string, opts ...Option) (*baseClient, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	if parsed.Host == "" && parsed.Path != "" {
		// allow bare hosts like localhost:8080
		parsed, err = url.Parse("http://" + trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint: %w", err)
		}
	}
	b := &baseClient{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		retry:      DefaultRetry(),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b, nil
}

func (b *baseClient) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}

	trimmed := strings.TrimPrefix(path, "/")
	rawQuery := ""
	if idx := strings.Index(trimmed, "?"); idx != -1 {
		rawQuery = trimmed[idx+1:]
		trimmed = trimmed[:idx]
	}

	clone := *b.baseURL
	if !strings.HasSuffix(clone.Path, "/") {
		clone.Path += "/"
	}
	basePath := strings.TrimSuffix(clone.Path, "/")
	if trimmed != "" {
		basePath = basePath + "/" + trimmed
	}
	clone.Path = basePath
	clone.RawQuery = rawQuery
	if rawQuery == "" {
		clone.RawQuery = ""
	}
	return clone.String()
}

func (b *baseClient) newJSONRequest(ctx context.Context, method, path string, payload interface{}) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
		body = buf
	}
	req, err := http.NewRequestWithContext(ctx, method, b.buildURL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", versionpkg.UserAgent())
	}
	return req, nil
}

// APIError is a non-2xx answer from the server. Error() keeps the historical
// "request failed: <body>" text so existing output does not change.
type APIError struct {
	StatusCode int
	Message    string
	// RetryAfter is the server's Retry-After, zero when it sent none.
	RetryAfter time.Duration
}

func (e *APIError) Error() string { return "request failed: " + e.Message }

func asAPIError(err error, target **APIError) bool { return errors.As(err, target) }

// streamClient is the HTTP client for long-lived responses: the same transport,
// no overall timeout.
func (b *baseClient) streamClient() httpDoer {
	if hc, ok := b.httpClient.(*http.Client); ok {
		clone := *hc
		clone.Timeout = 0
		return &clone
	}
	return b.httpClient
}

// IsStatus reports whether err is an *APIError with the given status.
func IsStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

// RetryOptions controls how transient failures are retried.
type RetryOptions struct {
	// Retries after the first attempt. Zero disables retrying.
	Retries int
	// BaseDelay is the first backoff step; it doubles each retry with full jitter.
	BaseDelay time.Duration
	MaxDelay  time.Duration
	// MaxRetryAfter is the longest Retry-After the client will wait for.
	MaxRetryAfter time.Duration

	sleep  func(context.Context, time.Duration) error
	random func() float64
}

// DefaultRetry is two retries starting at 200ms.
func DefaultRetry() RetryOptions {
	return RetryOptions{Retries: 2, BaseDelay: 200 * time.Millisecond, MaxDelay: 5 * time.Second, MaxRetryAfter: 30 * time.Second}
}

// WithRetry sets the retry policy; RetryOptions{} turns retrying off.
//
// Network errors and 429/502/503/504 are retried. GET, PUT and DELETE retry
// freely; a POST or PATCH only when it carries an Idempotency-Key header,
// because repeating it could otherwise duplicate the operation. A 429 without
// Retry-After (a daily quota) is not retried.
func WithRetry(r RetryOptions) Option {
	return func(b *baseClient) { b.retry = r }
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (r RetryOptions) backoff(attempt int) time.Duration {
	ceiling := r.BaseDelay << attempt
	if ceiling <= 0 || ceiling > r.MaxDelay {
		ceiling = r.MaxDelay
	}
	rnd := r.random
	if rnd == nil {
		rnd = rand.Float64 // #nosec G404 -- backoff jitter, not security
	}
	return time.Duration(rnd() * float64(ceiling))
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil && secs >= 0 {
		return time.Duration(secs * float64(time.Second))
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func safeToRepeat(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return req.Header.Get("Idempotency-Key") != ""
}

func (b *baseClient) do(req *http.Request, out interface{}) error {
	retry := b.retry
	sleep := retry.sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	repeatable := safeToRepeat(req)

	for attempt := 0; ; attempt++ {
		canRetry := repeatable && attempt < retry.Retries
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return err
			}
			req.Body = body
		}
		resp, err := b.httpClient.Do(req)
		if err != nil {
			if canRetry && req.Context().Err() == nil {
				if serr := sleep(req.Context(), retry.backoff(attempt)); serr != nil {
					return serr
				}
				continue
			}
			return err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			if out == nil || resp.StatusCode == http.StatusNoContent {
				return nil
			}
			dec := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)) // 4MB safety limit
			if err := dec.Decode(out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
			return nil
		}

		msg := readErrorBody(resp.Body)
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		status := resp.StatusCode
		_ = resp.Body.Close()
		if msg == "" {
			msg = resp.Status
		}

		if canRetry && (status == 429 || status == 502 || status == 503 || status == 504) {
			// A 429 without Retry-After is usually a daily quota: waiting seconds will not help.
			if status != 429 || (retryAfter > 0 && retryAfter <= retry.MaxRetryAfter) {
				wait := retryAfter
				if wait == 0 {
					wait = retry.backoff(attempt)
				}
				if serr := sleep(req.Context(), wait); serr != nil {
					return serr
				}
				continue
			}
		}
		return &APIError{StatusCode: status, Message: msg, RetryAfter: retryAfter}
	}
}

func readErrorBody(r io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(r, 4<<10)) // 4KB
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
