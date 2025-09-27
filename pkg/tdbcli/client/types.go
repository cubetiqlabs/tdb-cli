package client

import "time"

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
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
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

// DocumentBulkResponse is returned by bulk create endpoints.
type DocumentBulkResponse struct {
	Items []Document `json:"items"`
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
