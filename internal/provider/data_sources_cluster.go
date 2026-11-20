package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

type clusterHealthModel struct {
	Status           types.String `tfsdk:"status"`
	KnownNodes       types.Int64  `tfsdk:"known_nodes"`
	ConnectedNodes   types.Int64  `tfsdk:"connected_nodes"`
	StorageNodes     types.Int64  `tfsdk:"storage_nodes"`
	StorageNodesUp   types.Int64  `tfsdk:"storage_nodes_up"`
	Partitions       types.Int64  `tfsdk:"partitions"`
	PartitionsQuorum types.Int64  `tfsdk:"partitions_quorum"`
	PartitionsAllOk  types.Int64  `tfsdk:"partitions_all_ok"`
}

type clusterHealthDataSource struct {
	client *client.Client
}

func newClusterHealthDataSource() datasource.DataSource {
	return &clusterHealthDataSource{}
}

func (d *clusterHealthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_health"
}

func (d *clusterHealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *clusterHealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Health of the cluster as seen by the node behind the endpoint.",
		Attributes: map[string]schema.Attribute{
			"status":            schema.StringAttribute{Computed: true, MarkdownDescription: "`healthy`, `degraded` or `unavailable`."},
			"known_nodes":       schema.Int64Attribute{Computed: true},
			"connected_nodes":   schema.Int64Attribute{Computed: true},
			"storage_nodes":     schema.Int64Attribute{Computed: true},
			"storage_nodes_up":  schema.Int64Attribute{Computed: true},
			"partitions":        schema.Int64Attribute{Computed: true},
			"partitions_quorum": schema.Int64Attribute{Computed: true},
			"partitions_all_ok": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *clusterHealthDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	h, err := d.client.GetClusterHealth(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the cluster health", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &clusterHealthModel{
		Status:           types.StringValue(h.Status),
		KnownNodes:       types.Int64Value(h.KnownNodes),
		ConnectedNodes:   types.Int64Value(h.ConnectedNodes),
		StorageNodes:     types.Int64Value(h.StorageNodes),
		StorageNodesUp:   types.Int64Value(h.StorageNodesUp),
		Partitions:       types.Int64Value(h.Partitions),
		PartitionsQuorum: types.Int64Value(h.PartitionsQuorum),
		PartitionsAllOk:  types.Int64Value(h.PartitionsAllOk),
	})...)
}

type clusterNodeModel struct {
	ID                types.String `tfsdk:"id"`
	Hostname          types.String `tfsdk:"hostname"`
	Addr              types.String `tfsdk:"addr"`
	IsUp              types.Bool   `tfsdk:"is_up"`
	Draining          types.Bool   `tfsdk:"draining"`
	GarageVersion     types.String `tfsdk:"garage_version"`
	LastSeenSecsAgo   types.Int64  `tfsdk:"last_seen_secs_ago"`
	Zone              types.String `tfsdk:"zone"`
	Capacity          types.Int64  `tfsdk:"capacity"`
	Tags              types.List   `tfsdk:"tags"`
	DataAvailable     types.Int64  `tfsdk:"data_available"`
	DataTotal         types.Int64  `tfsdk:"data_total"`
	MetadataAvailable types.Int64  `tfsdk:"metadata_available"`
	MetadataTotal     types.Int64  `tfsdk:"metadata_total"`
}

type clusterStatusModel struct {
	LayoutVersion types.Int64        `tfsdk:"layout_version"`
	Nodes         []clusterNodeModel `tfsdk:"nodes"`
}

type clusterStatusDataSource struct {
	client *client.Client
}

func newClusterStatusDataSource() datasource.DataSource {
	return &clusterStatusDataSource{}
}

func (d *clusterStatusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_status"
}

func (d *clusterStatusDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *clusterStatusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nodes of the cluster, connected or not, with their role and disk space. Use it to find the node IDs for `garage_cluster_layout`.",
		Attributes: map[string]schema.Attribute{
			"layout_version": schema.Int64Attribute{Computed: true},
			"nodes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                 schema.StringAttribute{Computed: true, MarkdownDescription: "Full node ID."},
					"hostname":           schema.StringAttribute{Computed: true},
					"addr":               schema.StringAttribute{Computed: true, MarkdownDescription: "RPC address."},
					"is_up":              schema.BoolAttribute{Computed: true},
					"draining":           schema.BoolAttribute{Computed: true},
					"garage_version":     schema.StringAttribute{Computed: true},
					"last_seen_secs_ago": schema.Int64Attribute{Computed: true},
					"zone":               schema.StringAttribute{Computed: true},
					"capacity":           schema.Int64Attribute{Computed: true},
					"tags":               schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"data_available":     schema.Int64Attribute{Computed: true},
					"data_total":         schema.Int64Attribute{Computed: true},
					"metadata_available": schema.Int64Attribute{Computed: true},
					"metadata_total":     schema.Int64Attribute{Computed: true},
				}},
			},
		},
	}
}

func freeSpace(f *client.FreeSpace) (types.Int64, types.Int64) {
	if f == nil {
		return types.Int64Null(), types.Int64Null()
	}
	return types.Int64Value(f.Available), types.Int64Value(f.Total)
}

func (d *clusterStatusDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	status, err := d.client.GetClusterStatus(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the cluster status", err.Error())
		return
	}
	out := clusterStatusModel{LayoutVersion: types.Int64Value(status.LayoutVersion), Nodes: []clusterNodeModel{}}
	for _, n := range status.Nodes {
		node := clusterNodeModel{
			ID:              types.StringValue(n.ID),
			Hostname:        stringValue(n.Hostname),
			Addr:            stringValue(n.Addr),
			IsUp:            types.BoolValue(n.IsUp),
			Draining:        types.BoolValue(n.Draining),
			GarageVersion:   stringValue(n.GarageVersion),
			LastSeenSecsAgo: int64Value(n.LastSeenSecsAgo),
			Zone:            types.StringNull(),
			Capacity:        types.Int64Null(),
			Tags:            types.ListNull(types.StringType),
		}
		if n.Role != nil {
			node.Zone = types.StringValue(n.Role.Zone)
			node.Capacity = int64Value(n.Role.Capacity)
			node.Tags = stringsValue(n.Role.Tags)
		}
		node.DataAvailable, node.DataTotal = freeSpace(n.DataPartition)
		node.MetadataAvailable, node.MetadataTotal = freeSpace(n.MetadataPartition)
		out.Nodes = append(out.Nodes, node)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

type layoutRoleDataModel struct {
	ID               types.String `tfsdk:"id"`
	Zone             types.String `tfsdk:"zone"`
	Capacity         types.Int64  `tfsdk:"capacity"`
	Tags             types.List   `tfsdk:"tags"`
	StoredPartitions types.Int64  `tfsdk:"stored_partitions"`
	UsableCapacity   types.Int64  `tfsdk:"usable_capacity"`
}

type clusterLayoutDataModel struct {
	Version        types.Int64           `tfsdk:"version"`
	ZoneRedundancy types.Int64           `tfsdk:"zone_redundancy"`
	PartitionSize  types.Int64           `tfsdk:"partition_size"`
	StagedChanges  types.Bool            `tfsdk:"staged_changes"`
	Roles          []layoutRoleDataModel `tfsdk:"roles"`
}

type clusterLayoutDataSource struct {
	client *client.Client
}

func newClusterLayoutDataSource() datasource.DataSource {
	return &clusterLayoutDataSource{}
}

func (d *clusterLayoutDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_layout"
}

func (d *clusterLayoutDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *clusterLayoutDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The applied cluster layout, with the result of its computation.",
		Attributes: map[string]schema.Attribute{
			"version":         schema.Int64Attribute{Computed: true},
			"zone_redundancy": schema.Int64Attribute{Computed: true, MarkdownDescription: "Null means as many zones as possible."},
			"partition_size":  schema.Int64Attribute{Computed: true, MarkdownDescription: "Size of one partition, in bytes."},
			"staged_changes":  schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether changes are staged and not applied yet."},
			"roles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                schema.StringAttribute{Computed: true},
					"zone":              schema.StringAttribute{Computed: true},
					"capacity":          schema.Int64Attribute{Computed: true},
					"tags":              schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"stored_partitions": schema.Int64Attribute{Computed: true},
					"usable_capacity":   schema.Int64Attribute{Computed: true},
				}},
			},
		},
	}
}

func (d *clusterLayoutDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	layout, err := d.client.GetClusterLayout(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the cluster layout", err.Error())
		return
	}
	out := clusterLayoutDataModel{
		Version:        types.Int64Value(layout.Version),
		ZoneRedundancy: int64Value(layout.Parameters.ZoneRedundancy.AtLeast),
		PartitionSize:  types.Int64Value(layout.PartitionSize),
		StagedChanges:  types.BoolValue(len(layout.StagedRoleChanges) > 0 || layout.StagedParameters != nil),
		Roles:          []layoutRoleDataModel{},
	}
	for _, r := range layout.Roles {
		out.Roles = append(out.Roles, layoutRoleDataModel{
			ID:               types.StringValue(r.ID),
			Zone:             types.StringValue(r.Zone),
			Capacity:         int64Value(r.Capacity),
			Tags:             stringsValue(r.Tags),
			StoredPartitions: int64Value(r.StoredPartitions),
			UsableCapacity:   int64Value(r.UsableCapacity),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

type nodeDataModel struct {
	Node           types.String `tfsdk:"node"`
	NodeID         types.String `tfsdk:"node_id"`
	GarageVersion  types.String `tfsdk:"garage_version"`
	RustVersion    types.String `tfsdk:"rust_version"`
	DBEngine       types.String `tfsdk:"db_engine"`
	GarageFeatures types.List   `tfsdk:"garage_features"`
}

type nodeDataSource struct {
	client *client.Client
}

func newNodeDataSource() datasource.DataSource {
	return &nodeDataSource{}
}

func (d *nodeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

func (d *nodeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromDataSource(req, resp)
}

func (d *nodeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Build information of one node.",
		Attributes: map[string]schema.Attribute{
			"node":            schema.StringAttribute{Optional: true, MarkdownDescription: "Full node ID, or `self` (the default) for the node behind the endpoint."},
			"node_id":         schema.StringAttribute{Computed: true},
			"garage_version":  schema.StringAttribute{Computed: true},
			"rust_version":    schema.StringAttribute{Computed: true},
			"db_engine":       schema.StringAttribute{Computed: true, MarkdownDescription: "Metadata database engine."},
			"garage_features": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		},
	}
}

func (d *nodeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg nodeDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	node := "self"
	if !cfg.Node.IsNull() {
		node = cfg.Node.ValueString()
	}
	info, err := d.client.GetNodeInfo(ctx, node)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the node", err.Error())
		return
	}
	cfg.NodeID = types.StringValue(info.NodeID)
	cfg.GarageVersion = types.StringValue(info.GarageVersion)
	cfg.RustVersion = types.StringValue(info.RustVersion)
	cfg.DBEngine = types.StringValue(info.DBEngine)
	cfg.GarageFeatures = stringsValue(info.GarageFeatures)
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
