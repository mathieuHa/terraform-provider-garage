package client

import (
	"context"
	"net/url"
)

type BucketPermissions struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
	Owner bool `json:"owner"`
}

type BucketKey struct {
	AccessKeyID        string            `json:"accessKeyId"`
	Name               string            `json:"name"`
	Permissions        BucketPermissions `json:"permissions"`
	BucketLocalAliases []string          `json:"bucketLocalAliases"`
}

type Quotas struct {
	MaxSize    *int64 `json:"maxSize"`
	MaxObjects *int64 `json:"maxObjects"`
}

type WebsiteConfig struct {
	IndexDocument string        `json:"indexDocument"`
	ErrorDocument *string       `json:"errorDocument,omitempty"`
	RoutingRules  []RoutingRule `json:"routingRules,omitempty"`
}

type RoutingRule struct {
	Condition *RoutingCondition `json:"Condition,omitempty"`
	Redirect  RoutingRedirect   `json:"Redirect"`
}

type RoutingCondition struct {
	HTTPErrorCodeReturnedEquals *int64  `json:"HttpErrorCodeReturnedEquals,omitempty"`
	KeyPrefixEquals             *string `json:"KeyPrefixEquals,omitempty"`
}

type RoutingRedirect struct {
	HostName             *string `json:"HostName,omitempty"`
	HTTPRedirectCode     *int64  `json:"HttpRedirectCode,omitempty"`
	Protocol             *string `json:"Protocol,omitempty"`
	ReplaceKeyPrefixWith *string `json:"ReplaceKeyPrefixWith,omitempty"`
	ReplaceKeyWith       *string `json:"ReplaceKeyWith,omitempty"`
}

type CorsRule struct {
	ID            *string  `json:"ID,omitempty"`
	AllowedOrigin []string `json:"AllowedOrigin"`
	AllowedMethod []string `json:"AllowedMethod"`
	AllowedHeader []string `json:"AllowedHeader,omitempty"`
	ExposeHeader  []string `json:"ExposeHeader,omitempty"`
	MaxAgeSeconds *int64   `json:"MaxAgeSeconds,omitempty"`
}

type LifecycleRule struct {
	ID                             *string                         `json:"ID,omitempty"`
	Status                         string                          `json:"Status"`
	Filter                         *LifecycleFilter                `json:"Filter,omitempty"`
	Expiration                     *LifecycleExpiration            `json:"Expiration,omitempty"`
	AbortIncompleteMultipartUpload *LifecycleAbortIncompleteUpload `json:"AbortIncompleteMultipartUpload,omitempty"`
}

type LifecycleFilter struct {
	And                   *LifecycleFilter `json:"And,omitempty"`
	Prefix                *string          `json:"Prefix,omitempty"`
	ObjectSizeGreaterThan *int64           `json:"ObjectSizeGreaterThan,omitempty"`
	ObjectSizeLessThan    *int64           `json:"ObjectSizeLessThan,omitempty"`
}

type LifecycleExpiration struct {
	Days *int64  `json:"Days,omitempty"`
	Date *string `json:"Date,omitempty"`
}

type LifecycleAbortIncompleteUpload struct {
	DaysAfterInitiation int64 `json:"DaysAfterInitiation"`
}

type Bucket struct {
	ID                             string          `json:"id"`
	Created                        string          `json:"created"`
	GlobalAliases                  []string        `json:"globalAliases"`
	WebsiteAccess                  bool            `json:"websiteAccess"`
	WebsiteConfig                  *WebsiteConfig  `json:"websiteConfig"`
	Keys                           []BucketKey     `json:"keys"`
	Objects                        int64           `json:"objects"`
	Bytes                          int64           `json:"bytes"`
	UnfinishedUploads              int64           `json:"unfinishedUploads"`
	UnfinishedMultipartUploads     int64           `json:"unfinishedMultipartUploads"`
	UnfinishedMultipartUploadParts int64           `json:"unfinishedMultipartUploadParts"`
	UnfinishedMultipartUploadBytes int64           `json:"unfinishedMultipartUploadBytes"`
	Quotas                         Quotas          `json:"quotas"`
	CorsRules                      []CorsRule      `json:"corsRules"`
	LifecycleRules                 []LifecycleRule `json:"lifecycleRules"`
}

type BucketLocalAlias struct {
	AccessKeyID string `json:"accessKeyId"`
	Alias       string `json:"alias"`
}

type BucketListItem struct {
	ID            string             `json:"id"`
	Created       string             `json:"created"`
	GlobalAliases []string           `json:"globalAliases"`
	LocalAliases  []BucketLocalAlias `json:"localAliases"`
}

type CreateBucketRequest struct {
	GlobalAlias *string `json:"globalAlias,omitempty"`
}

type WebsiteAccessUpdate struct {
	Enabled       bool          `json:"enabled"`
	IndexDocument *string       `json:"indexDocument,omitempty"`
	ErrorDocument *string       `json:"errorDocument,omitempty"`
	RoutingRules  []RoutingRule `json:"routingRules,omitempty"`
}

type UpdateBucketRequest struct {
	Quotas         *Quotas              `json:"quotas,omitempty"`
	WebsiteAccess  *WebsiteAccessUpdate `json:"websiteAccess,omitempty"`
	CorsRules      *[]CorsRule          `json:"corsRules,omitempty"`
	LifecycleRules *[]LifecycleRule     `json:"lifecycleRules,omitempty"`
}

func (c *Client) CreateBucket(ctx context.Context, req CreateBucketRequest) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "CreateBucket", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBucket(ctx context.Context, id string) (*Bucket, error) {
	var out Bucket
	if err := c.get(ctx, "GetBucketInfo", url.Values{"id": {id}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBucketByGlobalAlias(ctx context.Context, alias string) (*Bucket, error) {
	var out Bucket
	if err := c.get(ctx, "GetBucketInfo", url.Values{"globalAlias": {alias}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListBuckets(ctx context.Context) ([]BucketListItem, error) {
	var out []BucketListItem
	if err := c.get(ctx, "ListBuckets", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateBucket(ctx context.Context, id string, req UpdateBucketRequest) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "UpdateBucket", url.Values{"id": {id}}, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBucket(ctx context.Context, id string) error {
	return c.post(ctx, "DeleteBucket", url.Values{"id": {id}}, nil, nil)
}

type bucketAliasRequest struct {
	BucketID    string `json:"bucketId"`
	GlobalAlias string `json:"globalAlias,omitempty"`
	LocalAlias  string `json:"localAlias,omitempty"`
	AccessKeyID string `json:"accessKeyId,omitempty"`
}

func aliasRequest(bucketID, alias, accessKeyID string) bucketAliasRequest {
	if accessKeyID == "" {
		return bucketAliasRequest{BucketID: bucketID, GlobalAlias: alias}
	}
	return bucketAliasRequest{BucketID: bucketID, LocalAlias: alias, AccessKeyID: accessKeyID}
}

func (c *Client) AddBucketAlias(ctx context.Context, bucketID, alias, accessKeyID string) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "AddBucketAlias", nil, aliasRequest(bucketID, alias, accessKeyID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RemoveBucketAlias(ctx context.Context, bucketID, alias, accessKeyID string) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "RemoveBucketAlias", nil, aliasRequest(bucketID, alias, accessKeyID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type bucketKeyPermRequest struct {
	BucketID    string            `json:"bucketId"`
	AccessKeyID string            `json:"accessKeyId"`
	Permissions BucketPermissions `json:"permissions"`
}

func (c *Client) AllowBucketKey(ctx context.Context, bucketID, accessKeyID string, perms BucketPermissions) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "AllowBucketKey", nil, bucketKeyPermRequest{bucketID, accessKeyID, perms}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DenyBucketKey(ctx context.Context, bucketID, accessKeyID string, perms BucketPermissions) (*Bucket, error) {
	var out Bucket
	if err := c.post(ctx, "DenyBucketKey", nil, bucketKeyPermRequest{bucketID, accessKeyID, perms}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetBucketKeyPermissions(ctx context.Context, bucketID, accessKeyID string, perms BucketPermissions) (*Bucket, error) {
	deny := BucketPermissions{Read: !perms.Read, Write: !perms.Write, Owner: !perms.Owner}
	bucket, err := c.DenyBucketKey(ctx, bucketID, accessKeyID, deny)
	if err != nil {
		return nil, err
	}
	if perms.Read || perms.Write || perms.Owner {
		return c.AllowBucketKey(ctx, bucketID, accessKeyID, perms)
	}
	return bucket, nil
}

func (b *Bucket) Key(accessKeyID string) *BucketKey {
	for i := range b.Keys {
		if b.Keys[i].AccessKeyID == accessKeyID {
			return &b.Keys[i]
		}
	}
	return nil
}
