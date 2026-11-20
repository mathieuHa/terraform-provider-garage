package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var (
	_ resource.Resource                = (*clusterLayoutResource)(nil)
	_ resource.ResourceWithImportState = (*clusterLayoutResource)(nil)
)

type clusterLayoutResource struct {
	client *client.Client
}

type clusterLayoutModel struct {
	ID             types.String               `tfsdk:"id"`
	Roles          map[string]layoutRoleModel `tfsdk:"roles"`
	ZoneRedundancy types.Int64                `tfsdk:"zone_redundancy"`
	Version        types.Int64                `tfsdk:"version"`
}

type layoutRoleModel struct {
	Zone     types.String `tfsdk:"zone"`
	Capacity types.Int64  `tfsdk:"capacity"`
	Tags     types.Set    `tfsdk:"tags"`
}

func newClusterLayoutResource() resource.Resource {
	return &clusterLayoutResource{}
}

func (r *clusterLayoutResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_layout"
}

func (r *clusterLayoutResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromResource(req, resp)
}

func (r *clusterLayoutResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The cluster layout: which nodes store data, in which zone, with which capacity. Declare one per cluster.\n\n" +
			"The resource is authoritative: nodes in the layout but not in `roles` are removed from it. Changes are staged and applied as one new layout version; changes staged by someone else are reverted first.\n\n" +
			"Destroying the resource leaves the layout as it is, because removing every role would make the cluster unusable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Always `cluster_layout`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Roles keyed by full node ID (64 hexadecimal characters, see the `garage_cluster_status` data source).",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(bucketIDRegex, "must be a full node ID")),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"zone": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Zone of the node, usually a site or a failure domain.",
						},
						"capacity": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Storage capacity of the node, in bytes. Unset makes the node a gateway that stores no data.",
							Validators:          []validator.Int64{int64validator.AtLeast(1)},
						},
						"tags": schema.SetAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Free-form tags.",
						},
					},
				},
			},
			"zone_redundancy": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum number of zones each partition is replicated in. Unset means as many zones as possible.",
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Version of the applied layout.",
			},
		},
	}
}

func roleChanged(current client.LayoutRole, desired layoutRoleModel) bool {
	if current.Zone != desired.Zone.ValueString() {
		return true
	}
	if (current.Capacity == nil) != desired.Capacity.IsNull() {
		return true
	}
	if current.Capacity != nil && *current.Capacity != desired.Capacity.ValueInt64() {
		return true
	}
	want := setStrings(desired.Tags)
	have := slices.Clone(current.Tags)
	slices.Sort(want)
	slices.Sort(have)
	return !slices.Equal(want, have)
}

func setStrings(s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return []string{}
	}
	out := make([]string, 0, len(s.Elements()))
	for _, e := range s.Elements() {
		if v, ok := e.(types.String); ok {
			out = append(out, v.ValueString())
		}
	}
	return out
}

func layoutChanges(layout *client.ClusterLayout, m *clusterLayoutModel) client.UpdateClusterLayoutRequest {
	req := client.UpdateClusterLayoutRequest{Roles: []client.RoleChange{}}
	current := map[string]client.LayoutRole{}
	for _, role := range layout.Roles {
		current[role.ID] = role
	}
	ids := make([]string, 0, len(m.Roles))
	for id := range m.Roles {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		desired := m.Roles[id]
		if role, ok := current[id]; ok && !roleChanged(role, desired) {
			continue
		}
		req.Roles = append(req.Roles, client.RoleChange{
			ID:       id,
			Zone:     desired.Zone.ValueString(),
			Capacity: int64Ptr(desired.Capacity),
			Tags:     setStrings(desired.Tags),
		})
	}
	for _, role := range layout.Roles {
		if _, ok := m.Roles[role.ID]; !ok {
			req.Roles = append(req.Roles, client.RoleChange{ID: role.ID, Remove: true})
		}
	}
	want := int64Ptr(m.ZoneRedundancy)
	have := layout.Parameters.ZoneRedundancy.AtLeast
	if (want == nil) != (have == nil) || (want != nil && *want != *have) {
		req.Parameters = &client.LayoutParameters{ZoneRedundancy: client.ZoneRedundancy{AtLeast: want}}
	}
	return req
}

func (m *clusterLayoutModel) readFrom(layout *client.ClusterLayout) {
	prior := m.Roles
	m.ID = types.StringValue("cluster_layout")
	m.Version = types.Int64Value(layout.Version)
	m.ZoneRedundancy = int64Value(layout.Parameters.ZoneRedundancy.AtLeast)
	m.Roles = map[string]layoutRoleModel{}
	for _, role := range layout.Roles {
		tags := types.SetNull(types.StringType)
		p, known := prior[role.ID]
		if len(role.Tags) > 0 || (known && !p.Tags.IsNull()) {
			elems := make([]attr.Value, 0, len(role.Tags))
			for _, t := range role.Tags {
				elems = append(elems, types.StringValue(t))
			}
			tags = types.SetValueMust(types.StringType, elems)
		}
		m.Roles[role.ID] = layoutRoleModel{
			Zone:     types.StringValue(role.Zone),
			Capacity: int64Value(role.Capacity),
			Tags:     tags,
		}
	}
}

func (r *clusterLayoutResource) apply(ctx context.Context, m *clusterLayoutModel) error {
	layout, err := r.client.GetClusterLayout(ctx)
	if err != nil {
		return err
	}
	if len(layout.StagedRoleChanges) > 0 || layout.StagedParameters != nil {
		if err := r.client.RevertClusterLayout(ctx); err != nil {
			return err
		}
		if layout, err = r.client.GetClusterLayout(ctx); err != nil {
			return err
		}
	}

	changes := layoutChanges(layout, m)
	if len(changes.Roles) > 0 || changes.Parameters != nil {
		if _, err := r.client.UpdateClusterLayout(ctx, changes); err != nil {
			return err
		}
		applied, err := r.client.ApplyClusterLayout(ctx, layout.Version+1)
		if err != nil {
			_ = r.client.RevertClusterLayout(ctx)
			return err
		}
		layout = &applied.Layout
	}
	m.readFrom(layout)
	return nil
}

func (r *clusterLayoutResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterLayoutModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot apply the cluster layout", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterLayoutResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterLayoutModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	layout, err := r.client.GetClusterLayout(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the cluster layout", err.Error())
		return
	}
	state.readFrom(layout)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterLayoutResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clusterLayoutModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot apply the cluster layout", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterLayoutResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Cluster layout left unchanged",
		"garage_cluster_layout was removed from the state, but the cluster layout was not changed: removing every role would make the cluster unusable.")
}

func (r *clusterLayoutResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m := clusterLayoutModel{ID: types.StringValue("cluster_layout"), ZoneRedundancy: types.Int64Null(), Version: types.Int64Null()}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
