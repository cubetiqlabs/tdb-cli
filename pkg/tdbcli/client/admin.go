package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// AdminClient provides helpers for interacting with admin endpoints.
type AdminClient struct {
	*baseClient
	secret string
}

// NewAdminClient constructs a new admin client using the supplied endpoint and admin secret.
func NewAdminClient(endpoint, adminSecret string, opts ...Option) (*AdminClient, error) {
	base, err := newBase(endpoint, opts...)
	if err != nil {
		return nil, err
	}
	if adminSecret == "" {
		return nil, fmt.Errorf("admin secret is required")
	}
	return &AdminClient{baseClient: base, secret: adminSecret}, nil
}

func (c *AdminClient) authorize(req *http.Request) {
	req.Header.Set("X-Admin-Secret", c.secret)
}

// ListTenants retrieves all tenants in the system.
func (c *AdminClient) ListTenants(ctx context.Context) ([]Tenant, error) {
	req, err := c.newJSONRequest(ctx, http.MethodGet, "/admin/tenants", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var tenants []Tenant
	if err := c.do(req, &tenants); err != nil {
		return nil, err
	}
	return tenants, nil
}

// CreateTenant creates a tenant. When request.WithAPIKey is true the response will include GeneratedKey details.
func (c *AdminClient) CreateTenant(ctx context.Context, request CreateTenantRequest) (*Tenant, *GeneratedKey, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/admin/tenants", request)
	if err != nil {
		return nil, nil, err
	}
	c.authorize(req)
	// Determine response shape based on WithAPIKey
	if request.WithAPIKey {
		var resp CreateTenantResponse
		if err := c.do(req, &resp); err != nil {
			return nil, nil, err
		}
		if resp.Tenant == nil {
			return nil, nil, fmt.Errorf("tenant creation succeeded but response missing tenant payload")
		}
		key := &GeneratedKey{APIKey: resp.APIKey, Prefix: resp.Prefix, Scope: "tenant"}
		if resp.Error != "" {
			return resp.Tenant, key, errors.New(resp.Error)
		}
		return resp.Tenant, key, nil
	}
	var tenant Tenant
	if err := c.do(req, &tenant); err != nil {
		return nil, nil, err
	}
	return &tenant, nil, nil
}

// GenerateKey creates an API key for a tenant or application depending on the request payload.
func (c *AdminClient) GenerateKey(ctx context.Context, tenantID string, request CreateAPIKeyRequest) (*GeneratedKey, error) {
	path := fmt.Sprintf("/admin/tenants/%s/keys", url.PathEscape(tenantID))
	req, err := c.newJSONRequest(ctx, http.MethodPost, path, request)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var resp GeneratedKey
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListKeys returns all API keys for a tenant. appID is optional for filtering.
func (c *AdminClient) ListKeys(ctx context.Context, tenantID string, appID *string) ([]APIKey, error) {
	path := fmt.Sprintf("/admin/tenants/%s/keys", url.PathEscape(tenantID))
	if appID != nil && *appID != "" {
		path += "?app_id=" + url.QueryEscape(*appID)
	}
	req, err := c.newJSONRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	var keys []APIKey
	if err := c.do(req, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// RevokeKey revokes the API key with the given prefix.
func (c *AdminClient) RevokeKey(ctx context.Context, prefix string) error {
	path := fmt.Sprintf("/admin/keys/%s", url.PathEscape(prefix))
	req, err := c.newJSONRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	return c.do(req, nil)
}
