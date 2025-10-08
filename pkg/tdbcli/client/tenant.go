package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TenantClient interacts with tenant-scoped endpoints using API keys.
type TenantClient struct {
	*baseClient
	apiKey string
}

// NewTenantClient creates a tenant-scoped client.
func NewTenantClient(endpoint, apiKey string, opts ...Option) (*TenantClient, error) {
	base, err := newBase(endpoint, opts...)
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	return &TenantClient{baseClient: base, apiKey: apiKey}, nil
}

func (c *TenantClient) authorize(req *http.Request) {
	req.Header.Set("X-API-Key", c.apiKey)
}

func (c *TenantClient) applyAppScope(req *http.Request, appID string) {
	trimmed := strings.TrimSpace(appID)
	if trimmed != "" {
		req.Header.Set("X-App-ID", trimmed)
	}
}

// ListApplications returns applications for the tenant represented by the API key.
func (c *TenantClient) ListApplications(ctx context.Context) ([]Application, error) {
	req, err := c.newJSONRequest(ctx, http.MethodGet, "/api/applications", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var resp struct {
		Items []Application `json:"items"`
	}
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// ListCollections lists collections for the tenant, optionally scoped to an application.
func (c *TenantClient) ListCollections(ctx context.Context, appID string) ([]Collection, error) {
	path := "/api/collections"
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var cols []Collection
	if err := c.do(req, &cols); err != nil {
		return nil, err
	}
	return cols, nil
}

// CountCollections returns the count of collections for the tenant.
func (c *TenantClient) CountCollections(ctx context.Context, appID string) (int64, error) {
	path := "/api/collections/count"
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var resp struct {
		Count int64 `json:"count"`
	}
	if err := c.do(req, &resp); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

// GetCollection fetches a collection by name.
func (c *TenantClient) GetCollection(ctx context.Context, name, appID string) (*Collection, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var col Collection
	if err := c.do(req, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// CreateCollection provisions a new collection for the tenant.
func (c *TenantClient) CreateCollection(ctx context.Context, reqBody CreateCollectionRequest) (*Collection, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/api/collections", reqBody)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, reqBody.AppID)
	var col Collection
	if err := c.do(req, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// UpdateCollection updates an existing collection by name.
func (c *TenantClient) UpdateCollection(ctx context.Context, name, appID string, reqBody UpdateCollectionRequest) (*Collection, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodPut, path, reqBody)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var col Collection
	if err := c.do(req, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// DeleteCollection removes a collection by name.
func (c *TenantClient) DeleteCollection(ctx context.Context, name, appID string) error {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	return c.do(req, nil)
}

// ListDocuments retrieves documents for a collection with optional filters.
func (c *TenantClient) ListDocuments(ctx context.Context, collection string, params ListDocumentsParams) (*DocumentListResponse, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(params.AppID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if params.Limit > 0 {
		values.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		values.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Cursor != "" {
		values.Set("cursor", params.Cursor)
	}
	if params.IncludeDeleted {
		values.Set("include_deleted", "true")
	}
	if len(params.SelectFields) > 0 {
		values.Set("select", strings.Join(params.SelectFields, ","))
	}
	if params.SelectOnly {
		values.Set("select_only", "true")
	}
	if len(params.Sort) > 0 {
		values.Set("sort", strings.Join(params.Sort, ","))
	}
	for field, value := range params.Filters {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			values.Set("f."+trimmed, value)
		}
	}
	path := fmt.Sprintf("/api/collections/%s/documents", url.PathEscape(collection))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, params.AppID)
	var resp DocumentListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamExport streams documents via the NDJSON export endpoint. Caller is responsible for closing the returned ReadCloser.
// selectFields optional (projection). selectOnly when true excludes implicit metadata fields.
func (c *TenantClient) StreamExport(ctx context.Context, collection string, selectFields []string, selectOnly bool, cursor string, limit int, appID string) (io.ReadCloser, http.Header, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if limit != 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	if trimmed := strings.TrimSpace(cursor); trimmed != "" {
		values.Set("cursor", trimmed)
	}
	if len(selectFields) > 0 {
		values.Set("select", strings.Join(selectFields, ","))
	}
	if selectOnly {
		values.Set("select_only", "true")
	}
	path := fmt.Sprintf("/api/collections/%s/export", url.PathEscape(collection))
	if enc := values.Encode(); enc != "" {
		path += "?" + enc
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, nil, fmt.Errorf("export request failed: %s", resp.Status)
	}
	return resp.Body, resp.Header, nil
}

// CountDocuments returns the number of documents in a collection.
func (c *TenantClient) CountDocuments(ctx context.Context, collection, appID string) (int64, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s/documents/count", url.PathEscape(collection))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var resp struct {
		Count int64 `json:"count"`
	}
	if err := c.do(req, &resp); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

// GetDocument fetches a document by ID.
func (c *TenantClient) GetDocument(ctx context.Context, collection, id, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s/documents/%s", url.PathEscape(collection), url.PathEscape(id))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetDocumentByPrimaryKey fetches a document using its primary key value.
func (c *TenantClient) GetDocumentByPrimaryKey(ctx context.Context, collection, key, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s/documents/primary/%s", url.PathEscape(collection), url.PathEscape(key))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// CreateDocument inserts a new document into a collection.
func (c *TenantClient) CreateDocument(ctx context.Context, collection string, payload []byte, appID string) (*Document, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, fmt.Sprintf("/api/collections/%s/documents", url.PathEscape(collection)), jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpdateDocument replaces a document by ID.
func (c *TenantClient) UpdateDocument(ctx context.Context, collection, id string, payload []byte, appID string) (*Document, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPut, fmt.Sprintf("/api/collections/%s/documents/%s", url.PathEscape(collection), url.PathEscape(id)), jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// PatchDocument applies a partial update to a document.
func (c *TenantClient) PatchDocument(ctx context.Context, collection, id string, payload []byte, appID string) (*Document, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPatch, fmt.Sprintf("/api/collections/%s/documents/%s", url.PathEscape(collection), url.PathEscape(id)), jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// DeleteDocument soft-deletes a document.
func (c *TenantClient) DeleteDocument(ctx context.Context, collection, id, appID string) error {
	req, err := c.newJSONRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/collections/%s/documents/%s", url.PathEscape(collection), url.PathEscape(id)), nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	return c.do(req, nil)
}

// PurgeDocument permanently deletes a document.
func (c *TenantClient) PurgeDocument(ctx context.Context, collection, id string, confirm bool, appID string) error {
	values := url.Values{}
	if confirm {
		values.Set("confirm", "true")
	}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/collections/%s/documents/%s/purge", url.PathEscape(collection), url.PathEscape(id))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	return c.do(req, nil)
}

// BulkCreateDocuments inserts multiple documents in one request.
func (c *TenantClient) BulkCreateDocuments(ctx context.Context, collection string, payload []byte, appID string) (*DocumentBulkResponse, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, fmt.Sprintf("/api/collections/%s/documents/bulk", url.PathEscape(collection)), jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var resp DocumentBulkResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSavedQueries returns saved query documents stored under the tenant's saved_queries collection.
func (c *TenantClient) ListSavedQueries(ctx context.Context, appID string) ([]Document, error) {
	path := "/api/queries"
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var resp SavedQueryListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// GetSavedQuery fetches a saved query document by ID.
func (c *TenantClient) GetSavedQuery(ctx context.Context, id, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/%s", url.PathEscape(id))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetSavedQueryByName fetches a saved query document by its canonical name.
func (c *TenantClient) GetSavedQueryByName(ctx context.Context, name, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/name/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// CreateSavedQuery creates or upserts a saved query document.
func (c *TenantClient) CreateSavedQuery(ctx context.Context, payload []byte, appID string) (*Document, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/api/queries", jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// PutSavedQuery fully replaces (or creates) a saved query by name.
func (c *TenantClient) PutSavedQuery(ctx context.Context, name string, payload []byte, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/name/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodPut, path, jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// PatchSavedQuery performs a shallow merge into a saved query by name.
func (c *TenantClient) PatchSavedQuery(ctx context.Context, name string, payload []byte, appID string) (*Document, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/name/%s", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodPatch, path, jsonRaw(payload))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var doc Document
	if err := c.do(req, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ExecuteSavedQueryByID runs a saved query by its document ID.
func (c *TenantClient) ExecuteSavedQueryByID(ctx context.Context, id string, payload []byte, appID string) (*SavedQueryExecutionResult, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/%s/execute", url.PathEscape(id))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var body interface{}
	if len(payload) > 0 {
		body = jsonRaw(payload)
	}
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var result SavedQueryExecutionResult
	if err := c.do(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteSavedQueryByName runs a saved query by its canonical name.
func (c *TenantClient) ExecuteSavedQueryByName(ctx context.Context, name string, payload []byte, appID string) (*SavedQueryExecutionResult, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	path := fmt.Sprintf("/api/queries/name/%s/execute", url.PathEscape(name))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var body interface{}
	if len(payload) > 0 {
		body = jsonRaw(payload)
	}
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var result SavedQueryExecutionResult
	if err := c.do(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteSavedQueryByID deletes or purges a saved query document by ID.
func (c *TenantClient) DeleteSavedQueryByID(ctx context.Context, id string, purge bool, appID string, confirm bool) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("saved query id is required")
	}
	if purge {
		return c.PurgeDocument(ctx, "saved_queries", id, confirm, appID)
	}
	return c.DeleteDocument(ctx, "saved_queries", id, appID)
}

// DeleteSavedQueryByName deletes or purges a saved query document identified by name.
func (c *TenantClient) DeleteSavedQueryByName(ctx context.Context, name string, purge bool, appID string, confirm bool) error {
	canonical := strings.ToLower(strings.TrimSpace(name))
	if canonical == "" {
		return fmt.Errorf("saved query name is required")
	}
	doc, err := c.GetSavedQueryByName(ctx, canonical, appID)
	if err != nil {
		return err
	}
	return c.DeleteSavedQueryByID(ctx, doc.ID, purge, appID, confirm)
}

// ReportQuery executes the reporting endpoint for ad-hoc aggregations.
func (c *TenantClient) ReportQuery(ctx context.Context, params ReportQueryParams) (*ReportQueryResponse, error) {
	collection := strings.TrimSpace(params.Collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	payload := make(map[string]any)
	if params.Body != nil {
		for key, value := range params.Body {
			if strings.EqualFold(key, "collection") {
				continue
			}
			payload[key] = value
		}
	}
	payload["collection"] = collection
	if params.Limit > 0 {
		if _, ok := payload["limit"]; !ok {
			payload["limit"] = params.Limit
		}
	}
	if params.Limit == -1 { // full scan sentinel
		payload["limit"] = -1
	}
	if params.Offset > 0 {
		if _, ok := payload["offset"]; !ok {
			payload["offset"] = params.Offset
		}
	}
	if trimmed := strings.TrimSpace(params.Cursor); trimmed != "" {
		if _, ok := payload["cursor"]; !ok {
			payload["cursor"] = trimmed
		}
	}
	if len(params.SelectFields) > 0 {
		if _, ok := payload["select"]; !ok {
			payload["select"] = params.SelectFields
		}
	}
	if params.SelectOnly {
		if _, ok := payload["selectOnly"]; !ok {
			payload["selectOnly"] = true
		}
	}
	encodedBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	if trimmed := strings.TrimSpace(params.AppID); trimmed != "" {
		values.Set("app_id", trimmed)
	}
	if params.Limit > 0 {
		values.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Limit == -1 {
		values.Set("limit", "-1")
	}
	if params.Offset > 0 {
		values.Set("offset", strconv.Itoa(params.Offset))
	}
	if trimmed := strings.TrimSpace(params.Cursor); trimmed != "" {
		values.Set("cursor", trimmed)
	}
	if len(params.SelectFields) > 0 {
		values.Set("select", strings.Join(params.SelectFields, ","))
	}
	if params.SelectOnly {
		values.Set("select_only", "true")
	}

	path := "/api/query"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, jsonRaw(encodedBody))
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, params.AppID)

	var resp ReportQueryResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAuditLogs retrieves audit log entries for the tenant with optional filters.
func (c *TenantClient) ListAuditLogs(ctx context.Context, params ListAuditLogsParams) ([]AuditLog, error) {
	values := url.Values{}
	if params.Limit > 0 {
		values.Set("limit", strconv.Itoa(params.Limit))
	}
	if trimmed := strings.TrimSpace(params.CollectionID); trimmed != "" {
		values.Set("collection", trimmed)
	}
	if trimmed := strings.TrimSpace(params.DocumentID); trimmed != "" {
		values.Set("document_id", trimmed)
	}
	if trimmed := strings.TrimSpace(params.Operation); trimmed != "" {
		values.Set("operation", strings.ToLower(trimmed))
	}
	if trimmed := strings.TrimSpace(params.Actor); trimmed != "" {
		values.Set("actor", trimmed)
	}
	if params.Since != nil && !params.Since.IsZero() {
		values.Set("since", params.Since.UTC().Format(time.RFC3339))
	}
	if params.Until != nil && !params.Until.IsZero() {
		values.Set("until", params.Until.UTC().Format(time.RFC3339))
	}
	if len(params.Sort) > 0 {
		values.Set("sort", strings.Join(params.Sort, ","))
	}
	path := "/api/audit"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, params.AppID)
	var resp AuditLogListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// AuthStatus retrieves the authentication context for the current API key via /api/me.
func (c *TenantClient) AuthStatus(ctx context.Context, appID string) (*AuthStatus, error) {
	path := "/api/me"
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	c.applyAppScope(req, appID)
	var status AuthStatus
	if err := c.do(req, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

type jsonRaw []byte

func (r jsonRaw) MarshalJSON() ([]byte, error) {
	return r, nil
}

// CreateApplication provisions an application for the tenant.
func (c *TenantClient) CreateApplication(ctx context.Context, request CreateApplicationRequest) (*Application, *GeneratedKey, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/api/applications", request)
	if err != nil {
		return nil, nil, err
	}
	c.authorize(req)
	if request.WithAPIKey {
		var resp CreateApplicationResponse
		if err := c.do(req, &resp); err != nil {
			return nil, nil, err
		}
		if resp.App == nil {
			return nil, nil, fmt.Errorf("application creation succeeded but response missing app payload")
		}
		key := &GeneratedKey{APIKey: resp.APIKey, Prefix: resp.Prefix, Scope: "application"}
		if resp.Error != "" {
			return resp.App, key, errors.New(resp.Error)
		}
		return resp.App, key, nil
	}
	var app Application
	if err := c.do(req, &app); err != nil {
		return nil, nil, err
	}
	return &app, nil, nil
}

// GetApplication fetches a single application by ID.
func (c *TenantClient) GetApplication(ctx context.Context, id string) (*Application, error) {
	path := fmt.Sprintf("/api/applications/%s", url.PathEscape(id))
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var app Application
	if err := c.do(req, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// ===== SNAPSHOT OPERATIONS =====

// ListSnapshots retrieves snapshots for the tenant
func (c *TenantClient) ListSnapshots(ctx context.Context, collectionID string, limit, offset int) ([]Snapshot, error) {
	query := url.Values{}
	if collectionID != "" {
		query.Set("collection_id", collectionID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}

	path := "/api/snapshots"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	var resp SnapshotListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// GetSnapshot retrieves a single snapshot by ID
func (c *TenantClient) GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error) {
	path := fmt.Sprintf("/api/snapshots/%s", url.PathEscape(snapshotID))
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	var snapshot Snapshot
	if err := c.do(req, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// CreateSnapshot creates a new snapshot
func (c *TenantClient) CreateSnapshot(ctx context.Context, request CreateSnapshotRequest) (*Snapshot, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/api/snapshots", request)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	var snapshot Snapshot
	if err := c.do(req, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// RestoreSnapshot restores a snapshot
func (c *TenantClient) RestoreSnapshot(ctx context.Context, snapshotID string, request RestoreSnapshotRequest) (*RestoreSnapshotResponse, error) {
	path := fmt.Sprintf("/api/snapshots/%s/restore", url.PathEscape(snapshotID))
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, request)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	var resp RestoreSnapshotResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSnapshot deletes a snapshot
func (c *TenantClient) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	path := fmt.Sprintf("/api/snapshots/%s", url.PathEscape(snapshotID))
	req, err := c.newJSONRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)

	return c.do(req, nil)
}
