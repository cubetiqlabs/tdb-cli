package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestVerifyCallsReadTheServerVerdict(t *testing.T) {
	c, _, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/audit/verify":
			fmt.Fprint(w, `{"ok":false,"checked":4,"unsealed":1,"broken_seq":3,"reason":"entry content does not match its hash"}`)
		case "/api/snapshots/snap%201/verify", "/api/snapshots/snap 1/verify":
			if r.Method != http.MethodPost {
				t.Errorf("snapshot verify must be a POST, got %s", r.Method)
			}
			fmt.Fprint(w, `{"snapshot_id":"snap 1","valid":false,"checksum_recorded":true,"checksum_matches":false,"problems":["checksum mismatch"]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}, RetryOptions{})

	a, err := c.VerifyAudit(context.Background())
	if err != nil || a.OK || a.BrokenSeq != 3 {
		t.Fatalf("audit verdict %+v err=%v", a, err)
	}
	s, err := c.VerifySnapshot(context.Background(), "snap 1")
	if err != nil || s.Valid || s.ChecksumMatches == nil || *s.ChecksumMatches {
		t.Fatalf("snapshot verdict %+v err=%v", s, err)
	}
}

func sseServer(t *testing.T, streams []string, seen *[]string, mu *sync.Mutex) *TenantClient {
	t.Helper()
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*seen = append(*seen, r.URL.RawQuery+"|"+r.Header.Get("X-API-Key"))
		i := n
		n++
		mu.Unlock()
		if r.URL.Path != "/events" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if i >= len(streams) {
			<-r.Context().Done() // hold the last connection open
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streams[i])
	}))
	t.Cleanup(srv.Close)
	c, err := NewTenantClient(srv.URL, "secret", WithRetry(RetryOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func event(seq int, typ, id string) string {
	return fmt.Sprintf("id: %d\nevent: %s\ndata: {\"seq\":%d,\"type\":%q,\"collection\":\"orders\",\"id\":%q}\n\n", seq, typ, seq, typ, id)
}

func TestWatchReconnectsAndResumesAfterTheLastSeq(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	c := sseServer(t, []string{
		": subscribed\n\n" + event(5, "create", "a") + event(6, "update", "a"),
		event(7, "delete", "a"),
	}, &seen, &mu)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var got []ChangeEvent
	err := c.Watch(ctx, "orders", "", nil, 10*time.Millisecond, func(ev ChangeEvent) error {
		got = append(got, ev)
		if ev.Seq == 7 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Watch should return nil when its context ends, got %v", err)
	}
	if len(got) != 3 || got[0].Seq != 5 || got[1].Type != "update" || got[2].Type != "delete" {
		t.Fatalf("events %+v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) < 2 {
		t.Fatalf("expected a reconnect, saw %v", seen)
	}
	if !strings.HasPrefix(seen[0], "collection=orders|secret") || strings.Contains(seen[0], "since") {
		t.Fatalf("first connect: %q", seen[0])
	}
	if !strings.Contains(seen[1], "since=6") {
		t.Fatalf("reconnect must resume after the last seq: %q", seen[1])
	}
}

func TestWatchStartsFromTheGivenCursorAndReportsLagged(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	c := sseServer(t, []string{"event: lagged\ndata: {\"type\":\"lagged\",\"seq\":40}\n\n" + event(41, "create", "x")}, &seen, &mu)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	since := uint64(40)
	var types []string
	_ = c.Watch(ctx, "orders", "", &since, 10*time.Millisecond, func(ev ChangeEvent) error {
		types = append(types, ev.Type)
		if ev.Seq == 41 {
			cancel()
		}
		return nil
	})
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(seen[0], "since=40") {
		t.Fatalf("query %q", seen[0])
	}
	if fmt.Sprint(types) != "[lagged create]" {
		t.Fatalf("types %v", types)
	}
}

func TestWatchFailsFastOnAnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "collection not found", http.StatusNotFound)
	}))
	defer srv.Close()
	c, _ := NewTenantClient(srv.URL, "k")
	err := c.Watch(context.Background(), "nope", "", nil, 10*time.Millisecond, func(ChangeEvent) error { return nil })
	if !IsStatus(err, http.StatusNotFound) {
		t.Fatalf("want a 404 error and no endless retry, got %v", err)
	}
}

func TestWatchStopsWhenTheHandlerFails(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	c := sseServer(t, []string{event(1, "create", "a")}, &seen, &mu)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.Watch(ctx, "orders", "", nil, 10*time.Millisecond, func(ChangeEvent) error { return fmt.Errorf("stop") })
	if err == nil || err.Error() != "stop" {
		t.Fatalf("handler error should end the watch, got %v", err)
	}
}
