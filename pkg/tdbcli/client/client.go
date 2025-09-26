package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	versionpkg "cubetiqlabs/tinydb/pkg/tdbcli/version"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type baseClient struct {
	baseURL    *url.URL
	httpClient httpDoer
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
	// Ensure we preserve base path
	rel := strings.TrimPrefix(path, "/")
	clone := *b.baseURL
	if !strings.HasSuffix(clone.Path, "/") {
		clone.Path += "/"
	}
	clone.Path = strings.TrimSuffix(clone.Path, "/") + "/" + rel
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

func (b *baseClient) do(req *http.Request, out interface{}) error {
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := readErrorBody(resp.Body)
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("request failed: %s", msg)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)) // 4MB safety limit
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func readErrorBody(r io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(r, 4<<10)) // 4KB
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
