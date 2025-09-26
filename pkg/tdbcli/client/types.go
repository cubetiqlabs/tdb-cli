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
