package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Usage is the tenant's standing against its plan.
type Usage struct {
	Plan            string `json:"plan,omitempty"`
	Suspended       bool   `json:"suspended"`
	SuspendedReason string `json:"suspended_reason,omitempty"`
	Limits          struct {
		RequestsPerDay int64 `json:"requests_per_day"`
		StorageBytes   int64 `json:"storage_bytes"`
		Documents      int64 `json:"documents"`
		Collections    int64 `json:"collections"`
		Applications   int64 `json:"applications"`
		APIKeys        int64 `json:"api_keys"`
	} `json:"limits"`
	Usage struct {
		Documents    int64 `json:"documents"`
		StorageBytes int64 `json:"storage_bytes"`
	} `json:"usage"`
	Collections  int64 `json:"collections"`
	Applications int64 `json:"applications"`
	APIKeys      int64 `json:"api_keys"`
	Requests     []struct {
		Day      string `json:"day"`
		Requests int64  `json:"requests"`
	} `json:"requests"`
}

// Usage reads the plan, limits, usage and recent daily request counts.
func (c *TenantClient) Usage(ctx context.Context, days int) (*Usage, error) {
	path := "/api/usage"
	if days > 0 {
		path += "?days=" + strconv.Itoa(days)
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var out Usage
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AuditVerification is the verdict of walking the tenant's audit hash chain.
type AuditVerification struct {
	OK        bool   `json:"ok"`
	Checked   int64  `json:"checked"`
	FirstSeq  int64  `json:"first_seq,omitempty"`
	LastSeq   int64  `json:"last_seq,omitempty"`
	LastHash  string `json:"last_hash,omitempty"`
	Unsealed  int64  `json:"unsealed"`
	BrokenSeq int64  `json:"broken_seq,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// VerifyAudit checks the tenant's tamper-evident audit chain.
func (c *TenantClient) VerifyAudit(ctx context.Context) (*AuditVerification, error) {
	req, err := c.newJSONRequest(ctx, http.MethodGet, "/api/audit/verify", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var out AuditVerification
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SnapshotVerification is the dry-run restore verdict for a snapshot.
type SnapshotVerification struct {
	SnapshotID         string   `json:"snapshot_id"`
	CollectionID       string   `json:"collection_id"`
	Valid              bool     `json:"valid"`
	ChecksumRecorded   bool     `json:"checksum_recorded"`
	ChecksumMatches    *bool    `json:"checksum_matches,omitempty"`
	DocumentsRecorded  int      `json:"documents_recorded"`
	DocumentsInPayload int      `json:"documents_in_payload"`
	ExistingDocuments  int64    `json:"existing_documents"`
	Problems           []string `json:"problems,omitempty"`
}

// VerifySnapshot checks a snapshot's integrity without restoring it.
func (c *TenantClient) VerifySnapshot(ctx context.Context, snapshotID string) (*SnapshotVerification, error) {
	path := fmt.Sprintf("/api/snapshots/%s/verify", url.PathEscape(snapshotID))
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var out SnapshotVerification
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChangeEvent is one change delivered by Watch. Seq is the resume cursor.
type ChangeEvent struct {
	Seq        uint64    `json:"seq,omitempty"`
	Type       string    `json:"type"`
	Collection string    `json:"collection"`
	ID         string    `json:"id,omitempty"`
	Data       string    `json:"data,omitempty"`
	Key        string    `json:"key,omitempty"`
	Timestamp  time.Time `json:"ts,omitempty"`
}

// Watch streams changes to a collection over Server-Sent Events, calling
// handle for each until ctx ends or handle returns an error.
//
// since, when non-nil, resumes after that seq: the server replays everything
// recorded since, then continues live. If the connection drops, Watch
// reconnects (after reconnectDelay) resuming from the last seq it delivered,
// so no change is missed. It returns nil when ctx is cancelled and an error for
// a non-2xx answer, which retrying would not fix.
func (c *TenantClient) Watch(ctx context.Context, collection, appID string, since *uint64, reconnectDelay time.Duration, handle func(ChangeEvent) error) error {
	if reconnectDelay <= 0 {
		reconnectDelay = time.Second
	}
	var last *uint64
	if since != nil {
		v := *since
		last = &v
	}
	for {
		err := c.watchOnce(ctx, collection, appID, last, func(ev ChangeEvent) error {
			if ev.Seq > 0 && (last == nil || ev.Seq > *last) {
				v := ev.Seq
				last = &v
			}
			return handle(ev)
		})
		if ctx.Err() != nil {
			return nil
		}
		var apiErr *APIError
		if asAPIError(err, &apiErr) {
			return err
		}
		if err != nil && isHandlerError(err) {
			return err
		}
		if serr := sleepCtx(ctx, reconnectDelay); serr != nil {
			return nil
		}
	}
}

type handlerError struct{ err error }

func (h handlerError) Error() string { return h.err.Error() }
func (h handlerError) Unwrap() error { return h.err }

func isHandlerError(err error) bool { _, ok := err.(handlerError); return ok }

func (c *TenantClient) watchOnce(ctx context.Context, collection, appID string, since *uint64, handle func(ChangeEvent) error) error {
	q := url.Values{"collection": {collection}}
	if since != nil {
		q.Set("since", strconv.FormatUint(*since, 10))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL("/events?"+q.Encode()), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	c.authorize(req)
	c.applyAppScope(req, appID)

	// The shared client has a request timeout meant for short calls; a stream must outlive it.
	resp, err := c.streamClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := readErrorBody(resp.Body)
		if msg == "" {
			msg = resp.Status
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	var data []string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if len(data) > 0 {
				var ev ChangeEvent
				if err := json.Unmarshal([]byte(strings.Join(data, "\n")), &ev); err == nil {
					if herr := handle(ev); herr != nil {
						return handlerError{herr}
					}
				}
				data = data[:0]
			}
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	return sc.Err()
}

// Call sends an authenticated JSON request to a tenant API path and decodes the
// answer into out (nil to discard it). It is for endpoints without a typed
// method, and keeps the client's retry and error handling.
func (c *TenantClient) Call(ctx context.Context, method, path string, body, out interface{}) error {
	req, err := c.newJSONRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	c.authorize(req)
	return c.do(req, out)
}
