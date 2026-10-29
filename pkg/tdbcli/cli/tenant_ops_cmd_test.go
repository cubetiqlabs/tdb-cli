package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI executes the real root command against a stub server and returns what
// it printed plus its error.
func runCLI(t *testing.T, h http.HandlerFunc, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	base := []string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "--endpoint", srv.URL}
	root.SetArgs(append(base, args...))
	err := root.ExecuteContext(context.Background())
	return out.String(), err
}

func auth() []string { return []string{"--tenant", "t1", "--api-key", "k"} }

func TestAuditVerifyReportsOKAndFailsOnABrokenChain(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true,"checked":10,"first_seq":1,"last_seq":10,"last_hash":"abc123","unsealed":2}`)
	}
	out, err := runCLI(t, ok, append([]string{"tenant", "audit", "verify"}, auth()...)...)
	if err != nil || !strings.Contains(out, "OK: 10 entries verified") || !strings.Contains(out, "abc123") {
		t.Fatalf("out=%q err=%v", out, err)
	}

	broken := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":false,"checked":4,"unsealed":0,"broken_seq":3,"reason":"entry content does not match its hash"}`)
	}
	out, err = runCLI(t, broken, append([]string{"tenant", "audit", "verify"}, auth()...)...)
	if err == nil || !strings.Contains(out, "BROKEN at seq 3") {
		t.Fatalf("a broken chain must fail the command: out=%q err=%v", out, err)
	}
}

func TestAuditVerifyDetectsTruncationAgainstARecordedHash(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true,"checked":10,"first_seq":1,"last_seq":10,"last_hash":"abc123","unsealed":0}`)
	}
	if _, err := runCLI(t, h, append([]string{"tenant", "audit", "verify", "--expect-hash", "ABC123"}, auth()...)...); err != nil {
		t.Fatalf("matching hash (case-insensitive) should pass: %v", err)
	}
	if _, err := runCLI(t, h, append([]string{"tenant", "audit", "verify", "--expect-hash", "deadbeef"}, auth()...)...); err == nil {
		t.Fatal("a different last_hash means entries were removed or rewritten; must fail")
	}
}

func TestSnapshotVerifyExitCodeFollowsTheVerdict(t *testing.T) {
	valid := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/snapshots/snap-1/verify" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"snapshot_id":"snap-1","valid":true,"checksum_recorded":true,"checksum_matches":true,"documents_recorded":5,"documents_in_payload":5,"existing_documents":7}`)
	}
	out, err := runCLI(t, valid, append([]string{"tenant", "snapshots", "verify", "--snapshot", "snap-1"}, auth()...)...)
	if err != nil || !strings.Contains(out, "VALID snapshot snap-1") || !strings.Contains(out, "would delete 7 live documents") {
		t.Fatalf("out=%q err=%v", out, err)
	}

	invalid := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"snapshot_id":"snap-1","valid":false,"checksum_recorded":true,"checksum_matches":false,"problems":["checksum mismatch"]}`)
	}
	out, err = runCLI(t, invalid, append([]string{"tenant", "snapshots", "verify", "--snapshot", "snap-1"}, auth()...)...)
	if err == nil || !strings.Contains(out, "INVALID") || !strings.Contains(out, "MISMATCH") || !strings.Contains(out, "checksum mismatch") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestSnapshotVerifyRequiresTheSnapshotFlag(t *testing.T) {
	_, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {}, append([]string{"tenant", "snapshots", "verify"}, auth()...)...)
	if err == nil || !strings.Contains(err.Error(), "snapshot") {
		t.Fatalf("want a required-flag error, got %v", err)
	}
}

func TestUsageShowsLimitsAndUnlimited(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("days") != "7" {
			t.Errorf("days not forwarded: %s", r.URL.RawQuery)
		}
		fmt.Fprint(w, `{"plan":"Starter","suspended":false,"limits":{"documents":500,"requests_per_day":0},"usage":{"documents":123,"storage_bytes":2048},"collections":4,"applications":1,"api_keys":2,"requests":[{"day":"20261028","requests":812}]}`)
	}
	out, err := runCLI(t, h, append([]string{"tenant", "usage", "--days", "7"}, auth()...)...)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Starter", "documents", "123", "500", "unlimited", "812"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDocumentCreateSendsTheIdempotencyKey(t *testing.T) {
	var got string
	h := func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"d1"}`)
	}
	out, err := runCLI(t, h, append([]string{"tenant", "documents", "create", "items", "--data", `{"a":1}`, "--idempotency-key", "abc-123"}, auth()...)...)
	if err != nil || !strings.Contains(out, "Created document d1") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if got != "abc-123" {
		t.Fatalf("Idempotency-Key %q", got)
	}
}

func TestWatchPrintsChangesAndResumesFromSince(t *testing.T) {
	var query string
	h := func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "id: 11\nevent: update\ndata: {\"seq\":11,\"type\":\"update\",\"collection\":\"orders\",\"id\":\"o1\",\"ts\":\"2026-10-29T10:00:00Z\"}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := httptest.NewServer(http.HandlerFunc(h))
	defer srv.Close()
	root := NewRootCommand()
	pr := &lineSignal{target: "seq=11", done: cancel}
	root.SetOut(pr)
	root.SetErr(pr)
	root.SetArgs(append([]string{"--config", filepath.Join(t.TempDir(), "c.yaml"), "--endpoint", srv.URL, "tenant", "watch", "orders", "--since", "10"}, auth()...))
	if err := root.ExecuteContext(ctx); err != nil {
		t.Fatalf("watch should end cleanly when cancelled: %v", err)
	}
	if !strings.Contains(query, "since=10") || !strings.Contains(query, "collection=orders") {
		t.Fatalf("query %q", query)
	}
	if !strings.Contains(pr.String(), "UPDATE") || !strings.Contains(pr.String(), "seq=11") {
		t.Fatalf("output %q", pr.String())
	}
}

// lineSignal is an output buffer that cancels a context once it has seen target.
type lineSignal struct {
	bytes.Buffer
	target string
	done   func()
}

func (l *lineSignal) Write(p []byte) (int, error) {
	n, err := l.Buffer.Write(p)
	if strings.Contains(l.String(), l.target) {
		l.done()
	}
	return n, err
}
