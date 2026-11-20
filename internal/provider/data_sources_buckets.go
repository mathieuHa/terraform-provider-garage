package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

type localAliasModel struct {
	AccessKeyID types.String `tfsdk:"access_key_id"`
	Alias       types.String `tfsdk:"alias"`
}

type bucketKeyModel struct {
	AccessKeyID  types.String `tfsdk:"access_key_id"`
	Name         types.String `tfsdk:"name"`
	Read         types.Bool   `tfsdk:"read"`
	Write        types.Bool   `tfsdk:"write"`
	Owner        types.Bool   `tfsdk:"owner"`
	LocalAliases types.List   `tfsdk:"local_aliases"`
}

type bucketDataModel struct {
	ID                         types.String      `tfsdk:"id"`
	GlobalAlias                types.String      `tfsdk:"global_alias"`
	GlobalAliases              types.List        `tfsdk:"global_aliases"`
	LocalAliases               []localAliasModel `tfsdk:"local_aliases"`
	Created                    types.String      `tfsdk:"created"`
	Objects                    types.Int64       `tfsdk:"objects"`
	Bytes                      types.Int64       `tfsdk:"bytes"`
	UnfinishedUploads          types.Int64       `tfsdk:"unfinished_uploads"`
	UnfinishedMultipartUploads types.Int64       `tfsdk:"unfinished_multipart_uploads"`
	MaxSize                    types.Int64       `tfsdk:"max_size"`
	MaxObjects                 types.Int64       `tfsdk:"max_objects"`
	WebsiteEnabled             types.Bool        `tfsdk:"website_enabled"`
	Keys                       []bucketKeyModel  `tfsdk:"keys"`
}

type bucketDataSource struct {
	client *client.Client
}

func newBucketDataSource() datasource.DataSource {
	return &bucketDataSource{}
}

func (d *bucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *bucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *bucketDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("global_alias")),
	}
}

func (d *bucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a bucket by ID or global alias, with its usage and the keys that have access to it.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Bucket ID."},
			"global_alias":   schema.StringAttribute{Optional: true, MarkdownDescription: "Global alias to look up."},
			"global_aliases": schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "All global aliases of the bucket."},
			"local_aliases": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Local aliases of the bucket.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{Computed: true},
					"alias":         schema.StringAttribute{Computed: true},
				}},
			},
			"created":                      schema.StringAttribute{Computed: true, MarkdownDescription: "Creation date."},
			"objects":                      schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of objects."},
			"bytes":                        schema.Int64Attribute{Computed: true, MarkdownDescription: "Total size of the objects, in bytes."},
			"unfinished_uploads":           schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of unfinished uploads."},
			"unfinished_multipart_uploads": schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of unfinished multipart uploads."},
			"max_size":                     schema.Int64Attribute{Computed: true, MarkdownDescription: "Size quota, in bytes, if any."},
			"max_objects":                  schema.Int64Attribute{Computed: true, MarkdownDescription: "Object count quota, if any."},
			"website_enabled":              schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether website hosting is enabled."},
			"keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Keys with permissions on the bucket.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{Computed: true},
					"name":          schema.StringAttribute{Computed: true},
					"read":          schema.BoolAttribute{Computed: true},
					"write":         schema.BoolAttribute{Computed: true},
					"owner":         schema.BoolAttribute{Computed: true},
					"local_aliases": schema.ListAttribute{Computed: true, ElementType: types.StringType},
				}},
			},
		},
	}
}

func (d *bucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg bucketDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var bucket *client.Bucket
	var err error
	if !cfg.ID.IsNull() {
		bucket, err = d.client.GetBucket(ctx, cfg.ID.ValueString())
	} else {
		bucket, err = d.client.GetBucketByGlobalAlias(ctx, cfg.GlobalAlias.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the bucket", err.Error())
		return
	}

	cfg.ID = types.StringValue(bucket.ID)
	cfg.GlobalAliases = stringsValue(bucket.GlobalAliases)
	cfg.Created = types.StringValue(bucket.Created)
	cfg.Objects = types.Int64Value(bucket.Objects)
	cfg.Bytes = types.Int64Value(bucket.Bytes)
	cfg.UnfinishedUploads = types.Int64Value(bucket.UnfinishedUploads)
	cfg.UnfinishedMultipartUploads = types.Int64Value(bucket.UnfinishedMultipartUploads)
	cfg.MaxSize = int64Value(bucket.Quotas.MaxSize)
	cfg.MaxObjects = int64Value(bucket.Quotas.MaxObjects)
	cfg.WebsiteEnabled = types.BoolValue(bucket.WebsiteAccess)
	cfg.LocalAliases = []localAliasModel{}
	cfg.Keys = []bucketKeyModel{}
	for _, k := range bucket.Keys {
		for _, a := range k.BucketLocalAliases {
			cfg.LocalAliases = append(cfg.LocalAliases, localAliasModel{AccessKeyID: types.StringValue(k.AccessKeyID), Alias: types.StringValue(a)})
		}
		cfg.Keys = append(cfg.Keys, bucketKeyModel{
			AccessKeyID:  types.StringValue(k.AccessKeyID),
			Name:         types.StringValue(k.Name),
			Read:         types.BoolValue(k.Permissions.Read),
			Write:        types.BoolValue(k.Permissions.Write),
			Owner:        types.BoolValue(k.Permissions.Owner),
			LocalAliases: stringsValue(k.BucketLocalAliases),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

type bucketsItemModel struct {
	ID            types.String      `tfsdk:"id"`
	Created       types.String      `tfsdk:"created"`
	GlobalAliases types.List        `tfsdk:"global_aliases"`
	LocalAliases  []localAliasModel `tfsdk:"local_aliases"`
}

type bucketsDataModel struct {
	Buckets []bucketsItemModel `tfsdk:"buckets"`
}

type bucketsDataSource struct {
	client *client.Client
}

func newBucketsDataSource() datasource.DataSource {
	return &bucketsDataSource{}
}

func (d *bucketsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buckets"
}

func (d *bucketsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *bucketsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every bucket of the cluster.",
		Attributes: map[string]schema.Attribute{
			"buckets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":             schema.StringAttribute{Computed: true},
					"created":        schema.StringAttribute{Computed: true},
					"global_aliases": schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"local_aliases": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
							"access_key_id": schema.StringAttribute{Computed: true},
							"alias":         schema.StringAttribute{Computed: true},
						}},
					},
				}},
			},
		},
	}
}

func (d *bucketsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	buckets, err := d.client.ListBuckets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot list the buckets", err.Error())
		return
	}
	out := bucketsDataModel{Buckets: []bucketsItemModel{}}
	for _, b := range buckets {
		item := bucketsItemModel{
			ID:            types.StringValue(b.ID),
			Created:       types.StringValue(b.Created),
			GlobalAliases: stringsValue(b.GlobalAliases),
			LocalAliases:  []localAliasModel{},
		}
		for _, a := range b.LocalAliases {
			item.LocalAliases = append(item.LocalAliases, localAliasModel{AccessKeyID: types.StringValue(a.AccessKeyID), Alias: types.StringValue(a.Alias)})
		}
		out.Buckets = append(out.Buckets, item)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
