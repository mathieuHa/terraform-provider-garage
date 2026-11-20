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

type keyBucketModel struct {
	ID            types.String `tfsdk:"id"`
	GlobalAliases types.List   `tfsdk:"global_aliases"`
	LocalAliases  types.List   `tfsdk:"local_aliases"`
	Read          types.Bool   `tfsdk:"read"`
	Write         types.Bool   `tfsdk:"write"`
	Owner         types.Bool   `tfsdk:"owner"`
}

type keyDataModel struct {
	ID                types.String     `tfsdk:"id"`
	Name              types.String     `tfsdk:"name"`
	Created           types.String     `tfsdk:"created"`
	Expiration        types.String     `tfsdk:"expiration"`
	Expired           types.Bool       `tfsdk:"expired"`
	AllowCreateBucket types.Bool       `tfsdk:"allow_create_bucket"`
	Buckets           []keyBucketModel `tfsdk:"buckets"`
}

type keyDataSource struct {
	client *client.Client
}

func newKeyDataSource() datasource.DataSource {
	return &keyDataSource{}
}

func (d *keyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (d *keyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *keyDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *keyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an access key by ID or by name, with the buckets it can access. The secret is never returned.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Access key ID."},
			"name":                schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Name of the key. The lookup fails if several keys match."},
			"created":             schema.StringAttribute{Computed: true},
			"expiration":          schema.StringAttribute{Computed: true},
			"expired":             schema.BoolAttribute{Computed: true},
			"allow_create_bucket": schema.BoolAttribute{Computed: true},
			"buckets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":             schema.StringAttribute{Computed: true},
					"global_aliases": schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"local_aliases":  schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"read":           schema.BoolAttribute{Computed: true},
					"write":          schema.BoolAttribute{Computed: true},
					"owner":          schema.BoolAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *keyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg keyDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var key *client.Key
	var err error
	if !cfg.ID.IsNull() {
		key, err = d.client.GetKey(ctx, cfg.ID.ValueString(), false)
	} else {
		key, err = d.client.SearchKey(ctx, cfg.Name.ValueString())
		if err == nil && key.Name != cfg.Name.ValueString() {
			resp.Diagnostics.AddError("Cannot find the key", "No key is named "+cfg.Name.ValueString()+".")
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the key", err.Error())
		return
	}

	cfg.ID = types.StringValue(key.AccessKeyID)
	cfg.Name = types.StringValue(key.Name)
	cfg.Created = stringValue(key.Created)
	cfg.Expiration = stringValue(key.Expiration)
	cfg.Expired = types.BoolValue(key.Expired)
	cfg.AllowCreateBucket = types.BoolValue(key.Permissions.CreateBucket)
	cfg.Buckets = []keyBucketModel{}
	for _, b := range key.Buckets {
		cfg.Buckets = append(cfg.Buckets, keyBucketModel{
			ID:            types.StringValue(b.ID),
			GlobalAliases: stringsValue(b.GlobalAliases),
			LocalAliases:  stringsValue(b.LocalAliases),
			Read:          types.BoolValue(b.Permissions.Read),
			Write:         types.BoolValue(b.Permissions.Write),
			Owner:         types.BoolValue(b.Permissions.Owner),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

type keysItemModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Created    types.String `tfsdk:"created"`
	Expiration types.String `tfsdk:"expiration"`
	Expired    types.Bool   `tfsdk:"expired"`
}

type keysDataModel struct {
	Keys []keysItemModel `tfsdk:"keys"`
}

type keysDataSource struct {
	client *client.Client
}

func newKeysDataSource() datasource.DataSource {
	return &keysDataSource{}
}

func (d *keysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keys"
}

func (d *keysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *keysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every access key of the cluster, without secrets.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":         schema.StringAttribute{Computed: true},
					"name":       schema.StringAttribute{Computed: true},
					"created":    schema.StringAttribute{Computed: true},
					"expiration": schema.StringAttribute{Computed: true},
					"expired":    schema.BoolAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *keysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot list the keys", err.Error())
		return
	}
	out := keysDataModel{Keys: []keysItemModel{}}
	for _, k := range keys {
		out.Keys = append(out.Keys, keysItemModel{
			ID:         types.StringValue(k.ID),
			Name:       types.StringValue(k.Name),
			Created:    stringValue(k.Created),
			Expiration: stringValue(k.Expiration),
			Expired:    types.BoolValue(k.Expired),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

type adminTokenDataModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Scope      types.List   `tfsdk:"scope"`
	Created    types.String `tfsdk:"created"`
	Expiration types.String `tfsdk:"expiration"`
	Expired    types.Bool   `tfsdk:"expired"`
}

type adminTokenDataSource struct {
	client *client.Client
}

func newAdminTokenDataSource() datasource.DataSource {
	return &adminTokenDataSource{}
}

func (d *adminTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_token"
}

func (d *adminTokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *adminTokenDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.Conflicting(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *adminTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an admin token by ID or name, or the token the provider uses when neither is set. The secret is never returned.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Token ID. Null for the tokens of the daemon configuration."},
			"name":       schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Name of the token."},
			"scope":      schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Admin endpoints the token can call."},
			"created":    schema.StringAttribute{Computed: true},
			"expiration": schema.StringAttribute{Computed: true},
			"expired":    schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *adminTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg adminTokenDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var token *client.AdminToken
	var err error
	switch {
	case !cfg.ID.IsNull():
		token, err = d.client.GetAdminToken(ctx, cfg.ID.ValueString())
	case !cfg.Name.IsNull():
		token, err = d.client.SearchAdminToken(ctx, cfg.Name.ValueString())
	default:
		token, err = d.client.GetCurrentAdminToken(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the admin token", err.Error())
		return
	}

	cfg.ID = stringValue(token.ID)
	cfg.Name = types.StringValue(token.Name)
	cfg.Scope = stringsValue(token.Scope)
	cfg.Created = stringValue(token.Created)
	cfg.Expiration = stringValue(token.Expiration)
	cfg.Expired = types.BoolValue(token.Expired)
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
