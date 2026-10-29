package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type fakeSleep struct{ waits []time.Duration }

func (f *fakeSleep) fn(_ context.Context, d time.Duration) error {
	f.waits = append(f.waits, d)
	return nil
}

func testClient(t *testing.T, h http.HandlerFunc, retry RetryOptions) (*TenantClient, *fakeSleep, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	fs := &fakeSleep{}
	retry.sleep = fs.fn
	retry.random = func() float64 { return 0.5 }
	c, err := NewTenantClient(srv.URL, "key", WithRetry(retry))
	if err != nil {
		t.Fatal(err)
	}
	return c, fs, &calls
}

func policy() RetryOptions {
	return RetryOptions{Retries: 2, BaseDelay: 200 * time.Millisecond, MaxDelay: 5 * time.Second, MaxRetryAfter: 30 * time.Second}
}

func TestGetRetriesThenSucceeds(t *testing.T) {
	var n int32
	c, fs, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) < 3 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"suspended":false,"collections":2}`)
	}, policy())

	u, err := c.Usage(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if u.Collections != 2 {
		t.Fatalf("unexpected usage %+v", u)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Fatalf("want 3 attempts, got %d", got)
	}
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond} // half of 200ms, half of 400ms
	if fmt.Sprint(fs.waits) != fmt.Sprint(want) {
		t.Fatalf("backoff waits %v, want %v", fs.waits, want)
	}
}

func TestGivesUpAfterRetriesWithATypedError(t *testing.T) {
	c, _, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}, policy())

	_, err := c.Usage(context.Background(), 0)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("want *APIError 502, got %v", err)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Fatalf("want 3 attempts, got %d", got)
	}
}

func TestErrorTextIsUnchanged(t *testing.T) {
	c, _, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}, policy())
	_, err := c.Usage(context.Background(), 0)
	if err == nil || err.Error() != "request failed: nope" {
		t.Fatalf("existing output must not change, got %q", err)
	}
	if !IsStatus(err, http.StatusForbidden) {
		t.Fatal("IsStatus should recognise the 403")
	}
}

func TestPostWithoutIdempotencyKeyIsNeverRetried(t *testing.T) {
	c, fs, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}, policy())

	_, err := c.CreateDocument(context.Background(), "items", []byte(`{"a":1}`), "")
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(calls); got != 1 || len(fs.waits) != 0 {
		t.Fatalf("a POST without a key must not be repeated (attempts=%d waits=%v)", got, fs.waits)
	}
}

func TestPostWithIdempotencyKeyIsRetriedWithTheSameKeyAndBody(t *testing.T) {
	var n int32
	var keys, bodies []string
	c, _, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 64)
		k, _ := r.Body.Read(buf)
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		bodies = append(bodies, string(buf[:k]))
		if atomic.AddInt32(&n, 1) == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"d1"}`)
	}, policy())

	doc, err := c.CreateDocumentIdempotent(context.Background(), "items", []byte(`{"a":1}`), "", "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "d1" || atomic.LoadInt32(calls) != 2 {
		t.Fatalf("doc=%+v calls=%d", doc, atomic.LoadInt32(calls))
	}
	if keys[0] != "key-1" || keys[1] != "key-1" {
		t.Fatalf("the same key must be resent: %v", keys)
	}
	if bodies[0] == "" || bodies[0] != bodies[1] {
		t.Fatalf("the body must be resent intact: %q", bodies)
	}
}

func TestRetryAfterReplacesTheComputedDelay(t *testing.T) {
	var n int32
	c, fs, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "2")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `{}`)
	}, policy())
	if _, err := c.Usage(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if len(fs.waits) != 1 || fs.waits[0] != 2*time.Second {
		t.Fatalf("waits %v, want [2s]", fs.waits)
	}
}

func TestTooManyRequestsWithoutRetryAfterIsNotRetried(t *testing.T) {
	c, _, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "daily quota exceeded", http.StatusTooManyRequests)
	}, policy())
	_, err := c.Usage(context.Background(), 0)
	if !IsStatus(err, http.StatusTooManyRequests) {
		t.Fatalf("want 429, got %v", err)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Fatal("a daily quota will not clear in seconds; do not retry")
	}
}

func TestRetryAfterBeyondTheCapIsSurfacedNotWaitedFor(t *testing.T) {
	c, fs, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}, policy())
	_, err := c.Usage(context.Background(), 0)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != time.Hour {
		t.Fatalf("want RetryAfter 1h on the error, got %v", err)
	}
	if atomic.LoadInt32(calls) != 1 || len(fs.waits) != 0 {
		t.Fatal("must not wait an hour")
	}
}

func TestClientErrorsAreNotRetried(t *testing.T) {
	c, _, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusNotFound)
	}, policy())
	_, _ = c.Usage(context.Background(), 0)
	if atomic.LoadInt32(calls) != 1 {
		t.Fatal("4xx must not be retried")
	}
}

func TestRetryCanBeDisabled(t *testing.T) {
	c, _, calls := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}, RetryOptions{})
	_, _ = c.Usage(context.Background(), 0)
	if atomic.LoadInt32(calls) != 1 {
		t.Fatal("zero retries means one attempt")
	}
}
