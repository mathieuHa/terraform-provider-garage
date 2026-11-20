package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	_ resource.Resource                = (*adminTokenResource)(nil)
	_ resource.ResourceWithImportState = (*adminTokenResource)(nil)
)

type adminTokenResource struct {
	client *client.Client
}

type adminTokenModel struct {
	ID          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Scope       types.List        `tfsdk:"scope"`
	Expiration  timetypes.RFC3339 `tfsdk:"expiration"`
	SecretToken types.String      `tfsdk:"secret_token"`
	Created     types.String      `tfsdk:"created"`
	Expired     types.Bool        `tfsdk:"expired"`
}

func newAdminTokenResource() resource.Resource {
	return &adminTokenResource{}
}

func (r *adminTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_token"
}

func (r *adminTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromResource(req, resp)
}

func (r *adminTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An admin API token, restricted to a list of admin endpoints. Use it to give automation only the access it needs instead of the daemon's `admin_token`.\n\n" +
			"The secret is only returned when the token is created, and is stored in the state. An imported token has no secret.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Token ID, which is also the beginning of the secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the token.",
			},
			"scope": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Admin endpoints the token can call, for example `[\"GetClusterHealth\", \"ListBuckets\"]`, or `[\"*\"]` for all of them. " +
					"Granting `CreateAdminToken` or `UpdateAdminToken` amounts to granting `*`.",
				Validators: []validator.List{listvalidator.SizeAtLeast(1), listvalidator.UniqueValues()},
			},
			"expiration": schema.StringAttribute{
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
				MarkdownDescription: "Date after which the token stops working, in RFC 3339 format. Unset means the token never expires.",
			},
			"secret_token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Bearer token to use with the admin API.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation date of the token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expired": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the token has expired.",
			},
		},
	}
}

func (m *adminTokenModel) request(prior *adminTokenModel) (client.UpdateAdminTokenRequest, error) {
	req := client.UpdateAdminTokenRequest{Name: stringPtr(m.Name), Scope: stringSlice(m.Scope)}
	expiration, diags := rfc3339String(m.Expiration)
	if diags.HasError() {
		return req, fmt.Errorf("invalid expiration")
	}
	if expiration != nil {
		req.Expiration = expiration
	} else if prior != nil && !prior.Expiration.IsNull() {
		req.NeverExpires = true
	}
	return req, nil
}

func (m *adminTokenModel) readFrom(t *client.AdminToken) {
	if t.ID != nil {
		m.ID = types.StringValue(*t.ID)
	}
	m.Name = types.StringValue(t.Name)
	m.Scope = stringsValue(t.Scope)
	m.Expiration = keepEquivalentTime(m.Expiration, rfc3339Value(t.Expiration))
	m.Created = stringValue(t.Created)
	m.Expired = types.BoolValue(t.Expired)
	if t.SecretToken != "" {
		m.SecretToken = types.StringValue(t.SecretToken)
	} else if m.SecretToken.IsUnknown() {
		m.SecretToken = types.StringNull()
	}
}

func (r *adminTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adminTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := plan.request(nil)
	if err != nil {
		resp.Diagnostics.AddError("Invalid admin token configuration", err.Error())
		return
	}
	token, err := r.client.CreateAdminToken(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Cannot create the admin token", err.Error())
		return
	}
	plan.readFrom(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adminTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adminTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	token, err := r.client.GetAdminToken(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the admin token", err.Error())
		return
	}
	state.readFrom(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adminTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adminTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := plan.request(&state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid admin token configuration", err.Error())
		return
	}
	token, err := r.client.UpdateAdminToken(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Cannot update the admin token", err.Error())
		return
	}
	plan.SecretToken = state.SecretToken
	plan.readFrom(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adminTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adminTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAdminToken(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Cannot delete the admin token", err.Error())
	}
}

func (r *adminTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("expiration"), timetypes.NewRFC3339Null())...)
}
