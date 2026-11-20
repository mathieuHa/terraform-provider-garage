package client

import (
	"context"
	"net/url"
)

type KeyPermissions struct {
	CreateBucket bool `json:"createBucket"`
}

type KeyBucket struct {
	ID            string            `json:"id"`
	GlobalAliases []string          `json:"globalAliases"`
	LocalAliases  []string          `json:"localAliases"`
	Permissions   BucketPermissions `json:"permissions"`
}

type Key struct {
	AccessKeyID     string         `json:"accessKeyId"`
	Name            string         `json:"name"`
	Created         *string        `json:"created"`
	Expiration      *string        `json:"expiration"`
	Expired         bool           `json:"expired"`
	Permissions     KeyPermissions `json:"permissions"`
	Buckets         []KeyBucket    `json:"buckets"`
	SecretAccessKey *string        `json:"secretAccessKey"`
}

type KeyListItem struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Created    *string `json:"created"`
	Expiration *string `json:"expiration"`
	Expired    bool    `json:"expired"`
}

type UpdateKeyRequest struct {
	Name         *string         `json:"name,omitempty"`
	Allow        *KeyPermissions `json:"allow,omitempty"`
	Deny         *KeyPermissions `json:"deny,omitempty"`
	Expiration   *string         `json:"expiration,omitempty"`
	NeverExpires bool            `json:"neverExpires,omitempty"`
}

type ImportKeyRequest struct {
	AccessKeyID     string  `json:"accessKeyId"`
	SecretAccessKey string  `json:"secretAccessKey"`
	Name            *string `json:"name,omitempty"`
}

func (c *Client) CreateKey(ctx context.Context, req UpdateKeyRequest) (*Key, error) {
	var out Key
	if err := c.post(ctx, "CreateKey", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ImportKey(ctx context.Context, req ImportKeyRequest) (*Key, error) {
	var out Key
	if err := c.post(ctx, "ImportKey", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetKey(ctx context.Context, id string, showSecret bool) (*Key, error) {
	q := url.Values{"id": {id}}
	if showSecret {
		q.Set("showSecretKey", "true")
	}
	var out Key
	if err := c.get(ctx, "GetKeyInfo", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SearchKey(ctx context.Context, search string) (*Key, error) {
	var out Key
	if err := c.get(ctx, "GetKeyInfo", url.Values{"search": {search}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListKeys(ctx context.Context) ([]KeyListItem, error) {
	var out []KeyListItem
	if err := c.get(ctx, "ListKeys", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateKey(ctx context.Context, id string, req UpdateKeyRequest) (*Key, error) {
	var out Key
	if err := c.post(ctx, "UpdateKey", url.Values{"id": {id}}, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteKey(ctx context.Context, id string) error {
	return c.post(ctx, "DeleteKey", url.Values{"id": {id}}, nil, nil)
}
