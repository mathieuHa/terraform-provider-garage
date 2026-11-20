package client

import (
	"context"
	"net/url"
)

type AdminToken struct {
	ID          *string  `json:"id"`
	Name        string   `json:"name"`
	Created     *string  `json:"created"`
	Expiration  *string  `json:"expiration"`
	Expired     bool     `json:"expired"`
	Scope       []string `json:"scope"`
	SecretToken string   `json:"secretToken,omitempty"`
}

type UpdateAdminTokenRequest struct {
	Name         *string  `json:"name,omitempty"`
	Scope        []string `json:"scope,omitempty"`
	Expiration   *string  `json:"expiration,omitempty"`
	NeverExpires bool     `json:"neverExpires,omitempty"`
}

func (c *Client) CreateAdminToken(ctx context.Context, req UpdateAdminTokenRequest) (*AdminToken, error) {
	var out AdminToken
	if err := c.post(ctx, "CreateAdminToken", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAdminToken(ctx context.Context, id string) (*AdminToken, error) {
	var out AdminToken
	if err := c.get(ctx, "GetAdminTokenInfo", url.Values{"id": {id}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SearchAdminToken(ctx context.Context, search string) (*AdminToken, error) {
	var out AdminToken
	if err := c.get(ctx, "GetAdminTokenInfo", url.Values{"search": {search}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCurrentAdminToken(ctx context.Context) (*AdminToken, error) {
	var out AdminToken
	if err := c.get(ctx, "GetCurrentAdminTokenInfo", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListAdminTokens(ctx context.Context) ([]AdminToken, error) {
	var out []AdminToken
	if err := c.get(ctx, "ListAdminTokens", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateAdminToken(ctx context.Context, id string, req UpdateAdminTokenRequest) (*AdminToken, error) {
	var out AdminToken
	if err := c.post(ctx, "UpdateAdminToken", url.Values{"id": {id}}, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAdminToken(ctx context.Context, id string) error {
	return c.post(ctx, "DeleteAdminToken", url.Values{"id": {id}}, nil, nil)
}
