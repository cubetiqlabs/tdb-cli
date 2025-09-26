package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
