package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var (
	_ resource.Resource                = (*bucketPermissionResource)(nil)
	_ resource.ResourceWithImportState = (*bucketPermissionResource)(nil)
)

type bucketPermissionResource struct {
	client *client.Client
}

type bucketPermissionModel struct {
	ID          types.String `tfsdk:"id"`
	BucketID    types.String `tfsdk:"bucket_id"`
	AccessKeyID types.String `tfsdk:"access_key_id"`
	Read        types.Bool   `tfsdk:"read"`
	Write       types.Bool   `tfsdk:"write"`
	Owner       types.Bool   `tfsdk:"owner"`
}

func newBucketPermissionResource() resource.Resource {
	return &bucketPermissionResource{}
}

func (r *bucketPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_permission"
}

func (r *bucketPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromResource(req, resp)
}

func (r *bucketPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Permissions of an access key on a bucket. Garage has no policies: a key can read, write and own a whole bucket, nothing finer.\n\n" +
			"The resource is authoritative for the pair: permissions set to `false` are revoked. Destroying it revokes every permission of the key on the bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "`<bucket_id>/<access_key_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the bucket.",
				PlanModifiers:       replace,
			},
			"access_key_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the access key.",
				PlanModifiers:       replace,
			},
			"read": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Read objects and list the bucket.",
			},
			"write": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Write and delete objects.",
			},
			"owner": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Change the bucket settings through the S3 API (website, CORS, lifecycle) and delete the bucket.",
			},
		},
	}
}

func (m *bucketPermissionModel) permissions() client.BucketPermissions {
	return client.BucketPermissions{Read: m.Read.ValueBool(), Write: m.Write.ValueBool(), Owner: m.Owner.ValueBool()}
}

func (r *bucketPermissionResource) apply(ctx context.Context, m *bucketPermissionModel) error {
	_, err := r.client.SetBucketKeyPermissions(ctx, m.BucketID.ValueString(), m.AccessKeyID.ValueString(), m.permissions())
	m.ID = types.StringValue(m.BucketID.ValueString() + "/" + m.AccessKeyID.ValueString())
	return err
}

func (r *bucketPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot set the bucket permissions", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, err := r.client.GetBucket(ctx, state.BucketID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the bucket", err.Error())
		return
	}

	key := bucket.Key(state.AccessKeyID.ValueString())
	if key == nil {
		if _, err := r.client.GetKey(ctx, state.AccessKeyID.ValueString(), false); client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		key = &client.BucketKey{}
	}
	state.Read = types.BoolValue(key.Permissions.Read)
	state.Write = types.BoolValue(key.Permissions.Write)
	state.Owner = types.BoolValue(key.Permissions.Owner)
	state.ID = types.StringValue(state.BucketID.ValueString() + "/" + state.AccessKeyID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bucketPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bucketPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot update the bucket permissions", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all := client.BucketPermissions{Read: true, Write: true, Owner: true}
	if _, err := r.client.DenyBucketKey(ctx, state.BucketID.ValueString(), state.AccessKeyID.ValueString(), all); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Cannot revoke the bucket permissions", err.Error())
	}
}

func (r *bucketPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <bucket_id>/<access_key_id>.")
		return
	}
	m := bucketPermissionModel{
		ID:          types.StringValue(req.ID),
		BucketID:    types.StringValue(parts[0]),
		AccessKeyID: types.StringValue(parts[1]),
		Read:        types.BoolValue(false),
		Write:       types.BoolValue(false),
		Owner:       types.BoolValue(false),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
