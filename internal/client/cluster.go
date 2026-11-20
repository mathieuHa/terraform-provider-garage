package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type ZoneRedundancy struct {
	AtLeast *int64
}

func (z ZoneRedundancy) MarshalJSON() ([]byte, error) {
	if z.AtLeast == nil {
		return json.Marshal("maximum")
	}
	return json.Marshal(map[string]int64{"atLeast": *z.AtLeast})
}

func (z *ZoneRedundancy) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s != "maximum" {
			return fmt.Errorf("unknown zone redundancy %q", s)
		}
		z.AtLeast = nil
		return nil
	}
	var obj struct {
		AtLeast int64 `json:"atLeast"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	z.AtLeast = &obj.AtLeast
	return nil
}

type LayoutParameters struct {
	ZoneRedundancy ZoneRedundancy `json:"zoneRedundancy"`
}

type LayoutRole struct {
	ID               string   `json:"id"`
	Zone             string   `json:"zone"`
	Capacity         *int64   `json:"capacity"`
	Tags             []string `json:"tags"`
	StoredPartitions *int64   `json:"storedPartitions"`
	UsableCapacity   *int64   `json:"usableCapacity"`
}

type RoleChange struct {
	ID       string
	Remove   bool
	Zone     string
	Capacity *int64
	Tags     []string
}

func (r RoleChange) MarshalJSON() ([]byte, error) {
	if r.Remove {
		return json.Marshal(map[string]any{"id": r.ID, "remove": true})
	}
	tags := r.Tags
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(map[string]any{"id": r.ID, "zone": r.Zone, "capacity": r.Capacity, "tags": tags})
}

type ClusterLayout struct {
	Version           int64             `json:"version"`
	Roles             []LayoutRole      `json:"roles"`
	Parameters        LayoutParameters  `json:"parameters"`
	PartitionSize     int64             `json:"partitionSize"`
	StagedRoleChanges []json.RawMessage `json:"stagedRoleChanges"`
	StagedParameters  *LayoutParameters `json:"stagedParameters"`
}

type UpdateClusterLayoutRequest struct {
	Parameters *LayoutParameters `json:"parameters,omitempty"`
	Roles      []RoleChange      `json:"roles"`
}

type ApplyClusterLayoutResponse struct {
	Message []string      `json:"message"`
	Layout  ClusterLayout `json:"layout"`
}

func (c *Client) GetClusterLayout(ctx context.Context) (*ClusterLayout, error) {
	var out ClusterLayout
	if err := c.get(ctx, "GetClusterLayout", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateClusterLayout(ctx context.Context, req UpdateClusterLayoutRequest) (*ClusterLayout, error) {
	var out ClusterLayout
	if err := c.post(ctx, "UpdateClusterLayout", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ApplyClusterLayout(ctx context.Context, version int64) (*ApplyClusterLayoutResponse, error) {
	var out ApplyClusterLayoutResponse
	if err := c.post(ctx, "ApplyClusterLayout", nil, map[string]int64{"version": version}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RevertClusterLayout(ctx context.Context) error {
	return c.post(ctx, "RevertClusterLayout", nil, nil, nil)
}

type FreeSpace struct {
	Available int64 `json:"available"`
	Total     int64 `json:"total"`
}

type AssignedRole struct {
	Zone     string   `json:"zone"`
	Capacity *int64   `json:"capacity"`
	Tags     []string `json:"tags"`
}

type ClusterNode struct {
	ID                string        `json:"id"`
	IsUp              bool          `json:"isUp"`
	Draining          bool          `json:"draining"`
	Addr              *string       `json:"addr"`
	Hostname          *string       `json:"hostname"`
	GarageVersion     *string       `json:"garageVersion"`
	LastSeenSecsAgo   *int64        `json:"lastSeenSecsAgo"`
	DataPartition     *FreeSpace    `json:"dataPartition"`
	MetadataPartition *FreeSpace    `json:"metadataPartition"`
	Role              *AssignedRole `json:"role"`
}

type ClusterStatus struct {
	LayoutVersion int64         `json:"layoutVersion"`
	Nodes         []ClusterNode `json:"nodes"`
}

type ClusterHealth struct {
	Status           string `json:"status"`
	KnownNodes       int64  `json:"knownNodes"`
	ConnectedNodes   int64  `json:"connectedNodes"`
	StorageNodes     int64  `json:"storageNodes"`
	StorageNodesUp   int64  `json:"storageNodesUp"`
	Partitions       int64  `json:"partitions"`
	PartitionsQuorum int64  `json:"partitionsQuorum"`
	PartitionsAllOk  int64  `json:"partitionsAllOk"`
}

type NodeInfo struct {
	NodeID         string   `json:"nodeId"`
	GarageVersion  string   `json:"garageVersion"`
	RustVersion    string   `json:"rustVersion"`
	DBEngine       string   `json:"dbEngine"`
	GarageFeatures []string `json:"garageFeatures"`
}

type nodeInfoMulti struct {
	Success map[string]NodeInfo `json:"success"`
	Error   map[string]string   `json:"error"`
}

func (c *Client) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	var out ClusterStatus
	if err := c.get(ctx, "GetClusterStatus", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	var out ClusterHealth
	if err := c.get(ctx, "GetClusterHealth", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetNodeInfo(ctx context.Context, node string) (*NodeInfo, error) {
	var out nodeInfoMulti
	if err := c.get(ctx, "GetNodeInfo", url.Values{"node": {node}}, &out); err != nil {
		return nil, err
	}
	for id, msg := range out.Error {
		return nil, fmt.Errorf("garage GetNodeInfo: node %s: %s", id, msg)
	}
	for _, info := range out.Success {
		return &info, nil
	}
	return nil, fmt.Errorf("garage GetNodeInfo: no node answered for %q", node)
}
