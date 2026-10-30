package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type seenReq struct {
	method, path, query string
	body                map[string]any
}

// record captures each request and answers with the given JSON.
func record(seen *[]seenReq, answer string, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		*seen = append(*seen, seenReq{r.Method, r.URL.Path, r.URL.RawQuery, body})
		if status != 0 {
			w.WriteHeader(status)
		}
		fmt.Fprint(w, answer)
	}
}

func TestWebhookCreateSendsTheFlagsAndPrintsTheSecretOnce(t *testing.T) {
	var seen []seenReq
	out, err := runCLI(t, record(&seen, `{"id":"wh1","name":"orders","secret":"s3cr3t-value"}`, http.StatusCreated),
		append([]string{"tenant", "webhooks", "create", "--name", "orders", "--url", "https://example.com/hook",
			"--event", "document.created", "--event", "document.updated", "--collection", "orders",
			"--header", "Authorization=Bearer x", "--max-retries", "3"}, auth()...)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].method != "POST" || seen[0].path != "/api/webhooks" {
		t.Fatalf("requests: %+v", seen)
	}
	b := seen[0].body
	if fmt.Sprint(b["events"]) != "[document.created document.updated]" || fmt.Sprint(b["collection_ids"]) != "[orders]" {
		t.Fatalf("body %v", b)
	}
	if h, _ := b["headers"].(map[string]any); h["Authorization"] != "Bearer x" {
		t.Fatalf("headers %v", b["headers"])
	}
	if b["max_retries"] != float64(3) {
		t.Fatalf("max_retries %v", b["max_retries"])
	}
	if !strings.Contains(out, "s3cr3t-value") || !strings.Contains(out, "shown once") {
		t.Fatalf("the secret must be shown with its warning:\n%s", out)
	}
}

func TestWebhookCreateRejectsMalformedHeaderFlags(t *testing.T) {
	var seen []seenReq
	_, err := runCLI(t, record(&seen, `{}`, 0),
		append([]string{"tenant", "webhooks", "create", "--name", "n", "--url", "https://x.test", "--event", "e", "--header", "no-equals-sign"}, auth()...)...)
	if err == nil || len(seen) != 0 {
		t.Fatalf("a bad --header must fail before any request (err=%v, requests=%d)", err, len(seen))
	}
}

func TestWebhookListShowsATableAndNeverASecret(t *testing.T) {
	var seen []seenReq
	out, err := runCLI(t, record(&seen, `{"items":[{"id":"wh1","name":"orders","url":"https://x.test","events":["document.created"],"enabled":false,"total_deliveries":10,"successful_deliveries":9}],"pagination":{}}`, 0),
		append([]string{"tenant", "webhooks", "list", "--limit", "5", "--offset", "10"}, auth()...)...)
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].query != "limit=5&offset=10" {
		t.Fatalf("paging not forwarded: %q", seen[0].query)
	}
	for _, want := range []string{"wh1", "orders", "paused", "9/10"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWebhookUpdateNeedsSomethingToChangeAndPausesWithEnabledFalse(t *testing.T) {
	var seen []seenReq
	if _, err := runCLI(t, record(&seen, `{}`, 0), append([]string{"tenant", "webhooks", "update", "wh1"}, auth()...)...); err == nil || len(seen) != 0 {
		t.Fatal("update with no flags must fail without calling the server")
	}
	if _, err := runCLI(t, record(&seen, `{"id":"wh1"}`, 0), append([]string{"tenant", "webhooks", "update", "wh1", "--enabled=false"}, auth()...)...); err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "PUT" || seen[0].body["enabled"] != false || len(seen[0].body) != 1 {
		t.Fatalf("only the flags passed are sent: %+v", seen[0])
	}
}

func TestWebhookRotateRedeliverAndDeleteRoutes(t *testing.T) {
	var seen []seenReq
	out, err := runCLI(t, record(&seen, `{"id":"wh1","secret":"fresh"}`, 0), append([]string{"tenant", "webhooks", "rotate-secret", "wh1"}, auth()...)...)
	if err != nil || !strings.Contains(out, "fresh") {
		t.Fatalf("rotate: %q %v", out, err)
	}
	if _, err := runCLI(t, record(&seen, `{"id":"d2"}`, http.StatusAccepted), append([]string{"tenant", "webhooks", "redeliver", "d1"}, auth()...)...); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, record(&seen, ``, http.StatusNoContent), append([]string{"tenant", "webhooks", "delete", "wh1"}, auth()...)...); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, s := range seen {
		got = append(got, s.method+" "+s.path)
	}
	want := "POST /api/webhooks/wh1/rotate-secret,POST /api/webhooks/deliveries/d1/redeliver,DELETE /api/webhooks/wh1"
	if strings.Join(got, ",") != want {
		t.Fatalf("routes %v", got)
	}
}

func TestScheduleCreateBuildsTheRetentionPolicyFromFlags(t *testing.T) {
	var seen []seenReq
	out, err := runCLI(t, record(&seen, `{"id":"sch1","name":"nightly","next_run_at":"2026-11-01T03:00:00Z"}`, http.StatusCreated),
		append([]string{"tenant", "schedules", "create", "--collection-id", "c1", "--name", "nightly", "--cron", "0 3 * * *",
			"--keep-daily", "7", "--keep-weekly", "4", "--paused"}, auth()...)...)
	if err != nil {
		t.Fatal(err)
	}
	b := seen[0].body
	if b["cron_expression"] != "0 3 * * *" || b["enabled"] != false {
		t.Fatalf("body %v", b)
	}
	rp, _ := b["retention_policy"].(map[string]any)
	if len(rp) != 2 || rp["keep_daily"] != float64(7) || rp["keep_weekly"] != float64(4) {
		t.Fatalf("retention %v", rp)
	}
	if !strings.Contains(out, "2026-11-01T03:00:00Z") {
		t.Fatalf("output %q", out)
	}
}

func TestScheduleEnableDisableAndTrigger(t *testing.T) {
	var seen []seenReq
	if _, err := runCLI(t, record(&seen, `{"id":"sch1","enabled":false}`, 0), append([]string{"tenant", "schedules", "disable", "sch1"}, auth()...)...); err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "PUT" || seen[0].body["enabled"] != false {
		t.Fatalf("disable: %+v", seen[0])
	}

	out, err := runCLI(t, record(&seen, `{"id":"ex1","status":"success","snapshot_id":"snap1","document_count":42}`, 0), append([]string{"tenant", "schedules", "trigger", "sch1"}, auth()...)...)
	if err != nil || !strings.Contains(out, "snap1") || !strings.Contains(out, "42 documents") {
		t.Fatalf("trigger: %q %v", out, err)
	}
	// A failed run must make the command fail, so scripts notice.
	_, err = runCLI(t, record(&seen, `{"id":"ex2","status":"failed","error":"disk full"}`, 0), append([]string{"tenant", "schedules", "trigger", "sch1"}, auth()...)...)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("a failed backup must exit non-zero with the reason, got %v", err)
	}
}

func TestScheduleExecutionsListsNewestFirst(t *testing.T) {
	var seen []seenReq
	out, err := runCLI(t, record(&seen, `{"executions":[{"id":"old","status":"success","snapshot_id":"s1","started_at":"2026-10-01T03:00:00Z"},{"id":"new","status":"failed","error":"boom","started_at":"2026-10-02T03:00:00Z"}]}`, 0),
		append([]string{"tenant", "schedules", "executions", "sch1"}, auth()...)...)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(out, "new") > strings.Index(out, "old") || !strings.Contains(out, "boom") {
		t.Fatalf("output:\n%s", out)
	}
}
