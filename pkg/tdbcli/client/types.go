package client

import (
	"encoding/json"
	"time"
)

// Tenant represents a tenant payload returned by the admin API.
type Tenant struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	RateLimitPerMinute *int      `json:"rate_limit_per_minute"`
	RequestDailyLimit  *int      `json:"request_daily_limit"`
	StorageBytesLimit  *int64    `json:"storage_bytes_limit"`
}

// APIKey mirrors the admin API key response.
type APIKey struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	AppID       *string    `json:"app_id"`
	Prefix      string     `json:"prefix"`
	Description *string    `json:"description"`
	Scope       string     `json:"scope"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}

// Collection mirrors the collection resource.
type Collection struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	AppID           *string    `json:"app_id"`
	Name            string     `json:"name"`
	SchemaJSON      string     `json:"schema_json"`
	PrimaryKeyField string     `json:"primary_key_field"`
	PrimaryKeyType  string     `json:"primary_key_type"`
	PrimaryKeyAuto  bool       `json:"primary_key_auto"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
	DocumentCount   int64      `json:"document_count"`
	StorageBytes    int64      `json:"storage_bytes"`
}

// PrimaryKeySpec configures a collection primary key.
type PrimaryKeySpec struct {
	Field string `json:"field"`
	Type  string `json:"type"`
	Auto  *bool  `json:"auto,omitempty"`
}

// CreateCollectionRequest is the payload for provisioning a collection.
type CreateCollectionRequest struct {
	Name       string          `json:"name"`
	Schema     string          `json:"schema"`
	AppID      string          `json:"app_id,omitempty"`
	PrimaryKey *PrimaryKeySpec `json:"primary_key,omitempty"`
	Sync       *bool           `json:"sync,omitempty"`
}

// UpdateCollectionRequest updates schema/primary key for a collection.
type UpdateCollectionRequest struct {
	Schema     string          `json:"schema,omitempty"`
	PrimaryKey *PrimaryKeySpec `json:"primary_key,omitempty"`
}

// Document represents a stored document record.
type Document struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	CollectionID string     `json:"collection_id"`
	Key          string     `json:"key"`
	KeyNumeric   *int64     `json:"key_numeric"`
	Data         string     `json:"data"`
	Version      int64      `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
	DataSize     int64      `json:"data_size"`
}

// AuditLog captures history entries for document lifecycle events.
type AuditLog struct {
	ID              uint      `json:"id"`
	TenantID        string    `json:"tenant_id"`
	CollectionID    string    `json:"collection_id"`
	DocumentID      string    `json:"document_id"`
	DocumentVersion int64     `json:"document_version"`
	Operation       string    `json:"operation"`
	Actor           string    `json:"actor"`
	OldData         string    `json:"old_data"`
	NewData         string    `json:"new_data"`
	CreatedAt       time.Time `json:"created_at"`
	OldDataSize     int64     `json:"old_data_size"`
	NewDataSize     int64     `json:"new_data_size"`
	ChangeSize      int64     `json:"change_size"`
}

// DocumentPagination exposes pagination metadata for list endpoints.
type DocumentPagination struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Count  int64 `json:"count"`
}

// DocumentListResponse is returned by list endpoints.
type DocumentListResponse struct {
	Items      []Document         `json:"items"`
	Pagination DocumentPagination `json:"pagination"`
}

// AuditLogListResponse wraps audit log list responses.
type AuditLogListResponse struct {
	Items []AuditLog `json:"items"`
}

// DocumentBulkResponse is returned by bulk create endpoints.
type DocumentBulkResponse struct {
	Items []Document `json:"items"`
}

// ReportQueryResponse captures the payload returned by POST /api/query.
type ReportQueryResponse struct {
	Data       []map[string]any      `json:"data"`
	Pagination ReportQueryPagination `json:"pagination"`
}

// ReportQueryPagination describes pagination metadata for report queries.
type ReportQueryPagination struct {
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Total      int64  `json:"total"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// SavedQuery represents the inner payload of a saved query document.
type SavedQuery struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Collection string          `json:"collection,omitempty"`
	DSL        json.RawMessage `json:"dsl,omitempty"`
	SQL        string          `json:"sql,omitempty"`
}

// SavedQueryListResponse captures the saved query listing payload.
type SavedQueryListResponse struct {
	Items []Document `json:"items"`
}

// SavedQueryExecutionResult contains the result rows when executing a saved query.
type SavedQueryExecutionResult struct {
	Items []map[string]any `json:"items"`
}

// SavedQueryPatchRequest is used when partially updating a saved query by name.
type SavedQueryPatchRequest map[string]any

// AuthStatus represents the payload returned by GET /api/me.
type AuthStatus struct {
	TenantID   string     `json:"tenant_id"`
	TenantName string     `json:"tenant_name"`
	AppID      string     `json:"app_id"`
	AppName    string     `json:"app_name"`
	Status     string     `json:"status"`
	KeyPrefix  string     `json:"key_prefix,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
	Scope      string     `json:"scope,omitempty"`
}

// ListDocumentsParams configures document list queries.
type ListDocumentsParams struct {
	AppID          string
	Limit          int
	Offset         int
	Cursor         string
	IncludeDeleted bool
	SelectFields   []string
	Filters        map[string]string
	Sort           []string
}

// ReportQueryParams configures report query requests.
type ReportQueryParams struct {
	AppID        string
	Collection   string
	Limit        int
	Offset       int
	Cursor       string
	SelectFields []string
	Body         map[string]any
}

// ListAuditLogsParams configures audit log retrieval.
type ListAuditLogsParams struct {
	AppID        string
	Limit        int
	CollectionID string
	DocumentID   string
	Operation    string
	Actor        string
	Since        *time.Time
	Until        *time.Time
	Sort         []string
}

// Application represents the application resource exposed via tenant API.
type Application struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

// CreateTenantRequest is used when provisioning a new tenant.
type CreateTenantRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	WithAPIKey  bool   `json:"with_api_key,omitempty"`
}

// CreateTenantResponse is the response when WithAPIKey is enabled.
type CreateTenantResponse struct {
	Tenant *Tenant `json:"tenant"`
	APIKey string  `json:"api_key"`
	Prefix string  `json:"prefix"`
	Error  string  `json:"error,omitempty"`
}

// CreateAPIKeyRequest represents parameters for generating a key.
type CreateAPIKeyRequest struct {
	AppID       *string `json:"app_id,omitempty"`
	Description *string `json:"description,omitempty"`
}

// GeneratedKey wraps the response for newly generated keys.
type GeneratedKey struct {
	APIKey      string  `json:"api_key"`
	Prefix      string  `json:"prefix"`
	Description *string `json:"description"`
	Scope       string  `json:"scope"`
	AppID       *string `json:"app_id,omitempty"`
}

// CreateApplicationRequest captures application creation payloads.
type CreateApplicationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WithAPIKey  bool   `json:"with_api_key,omitempty"`
}

// CreateApplicationResponse is returned when requesting an API key for the new app.
type CreateApplicationResponse struct {
	App    *Application `json:"app"`
	APIKey string       `json:"api_key"`
	Prefix string       `json:"prefix"`
	Error  string       `json:"error,omitempty"`
}
