package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var (
	_ resource.Resource                = (*bucketAliasResource)(nil)
	_ resource.ResourceWithImportState = (*bucketAliasResource)(nil)
)

type bucketAliasResource struct {
	client *client.Client
}

type bucketAliasModel struct {
	ID          types.String `tfsdk:"id"`
	BucketID    types.String `tfsdk:"bucket_id"`
	Alias       types.String `tfsdk:"alias"`
	AccessKeyID types.String `tfsdk:"access_key_id"`
}

func newBucketAliasResource() resource.Resource {
	return &bucketAliasResource{}
}

func (r *bucketAliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_alias"
}

func (r *bucketAliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromResource(req, resp)
}

func (r *bucketAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "An additional alias of a bucket. A global alias is a bucket name visible to every key. A local alias, set with `access_key_id`, is a bucket name visible only to that key.\n\n" +
			"Do not manage the same global alias here and in `garage_bucket.global_alias`. Garage refuses to remove the last alias of a bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "`<bucket_id>/<alias>` for a global alias, `<bucket_id>/<access_key_id>/<alias>` for a local alias.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the bucket.",
				PlanModifiers:       replace,
			},
			"alias": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The alias.",
				PlanModifiers:       replace,
				Validators:          []validator.String{stringvalidator.RegexMatches(bucketNameRegex, "must be a valid S3 bucket name")},
			},
			"access_key_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Access key that owns the alias. Set for a local alias, unset for a global alias.",
				PlanModifiers:       replace,
			},
		},
	}
}

func (m *bucketAliasModel) setID() {
	if m.AccessKeyID.IsNull() {
		m.ID = types.StringValue(m.BucketID.ValueString() + "/" + m.Alias.ValueString())
		return
	}
	m.ID = types.StringValue(strings.Join([]string{m.BucketID.ValueString(), m.AccessKeyID.ValueString(), m.Alias.ValueString()}, "/"))
}

func (r *bucketAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.AddBucketAlias(ctx, plan.BucketID.ValueString(), plan.Alias.ValueString(), plan.AccessKeyID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot add the bucket alias", err.Error())
		return
	}
	plan.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketAliasModel
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

	if !bucketHasAlias(bucket, state.Alias.ValueString(), state.AccessKeyID.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}
	state.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func bucketHasAlias(bucket *client.Bucket, alias, accessKeyID string) bool {
	if accessKeyID == "" {
		return slices.Contains(bucket.GlobalAliases, alias)
	}
	key := bucket.Key(accessKeyID)
	return key != nil && slices.Contains(key.BucketLocalAliases, alias)
}

func (r *bucketAliasResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "Every attribute of garage_bucket_alias forces a replacement.")
}

func (r *bucketAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemoveBucketAlias(ctx, state.BucketID.ValueString(), state.Alias.ValueString(), state.AccessKeyID.ValueString())
	if err == nil || client.IsNotFound(err) {
		return
	}
	bucket, probeErr := r.client.GetBucket(ctx, state.BucketID.ValueString())
	if client.IsNotFound(probeErr) || (probeErr == nil && !bucketHasAlias(bucket, state.Alias.ValueString(), state.AccessKeyID.ValueString())) {
		return
	}
	resp.Diagnostics.AddError("Cannot remove the bucket alias",
		fmt.Sprintf("%s\n\nGarage refuses to remove the last alias of a bucket: give it another alias first, or delete the bucket.", err))
}

func (r *bucketAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	var m bucketAliasModel
	switch len(parts) {
	case 2:
		m = bucketAliasModel{BucketID: types.StringValue(parts[0]), Alias: types.StringValue(parts[1]), AccessKeyID: types.StringNull()}
	case 3:
		m = bucketAliasModel{BucketID: types.StringValue(parts[0]), AccessKeyID: types.StringValue(parts[1]), Alias: types.StringValue(parts[2])}
	default:
		resp.Diagnostics.AddError("Invalid import ID", "Expected <bucket_id>/<alias> or <bucket_id>/<access_key_id>/<alias>.")
		return
	}
	m.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), m.ID)...)
}
