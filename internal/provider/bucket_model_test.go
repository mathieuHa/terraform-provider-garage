package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExpandLifecycleFilter(t *testing.T) {
	for name, tc := range map[string]struct {
		filter *lifecycleFilterModel
		want   string
	}{
		"none":        {nil, `null`},
		"empty":       {&lifecycleFilterModel{Prefix: types.StringNull(), ObjectSizeGreaterThan: types.Int64Null(), ObjectSizeLessThan: types.Int64Null()}, `null`},
		"prefix only": {&lifecycleFilterModel{Prefix: types.StringValue("tmp/"), ObjectSizeGreaterThan: types.Int64Null(), ObjectSizeLessThan: types.Int64Null()}, `{"Prefix":"tmp/"}`},
		"wrapped":     {&lifecycleFilterModel{Prefix: types.StringValue("tmp/"), ObjectSizeGreaterThan: types.Int64Value(10), ObjectSizeLessThan: types.Int64Null()}, `{"And":{"Prefix":"tmp/","ObjectSizeGreaterThan":10}}`},
	} {
		t.Run(name, func(t *testing.T) {
			if got := mustJSON(t, expandLifecycleFilter(tc.filter)); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLifecycleRoundTrip(t *testing.T) {
	prefix := "tmp/"
	gt := int64(10)
	days := int64(7)
	id := "a"
	rules := []client.LifecycleRule{
		{ID: &id, Status: "Enabled", Filter: &client.LifecycleFilter{And: &client.LifecycleFilter{Prefix: &prefix, ObjectSizeGreaterThan: &gt}}, Expiration: &client.LifecycleExpiration{Days: &days}},
		{Status: "Disabled", AbortIncompleteMultipartUpload: &client.LifecycleAbortIncompleteUpload{DaysAfterInitiation: 2}},
	}
	flat := flattenLifecycleRules(nil, rules)
	if len(flat) != 2 || flat[0].Filter == nil || flat[0].Filter.Prefix.ValueString() != "tmp/" || flat[0].Filter.ObjectSizeGreaterThan.ValueInt64() != 10 {
		t.Fatalf("unexpected flattening %+v", flat)
	}
	if flat[1].Enabled.ValueBool() || flat[1].Filter != nil || flat[1].AbortIncompleteMultipartUploadDays.ValueInt64() != 2 {
		t.Fatalf("unexpected second rule %+v", flat[1])
	}
	if got, want := mustJSON(t, expandLifecycleRules(flat)), mustJSON(t, rules); got != want {
		t.Fatalf("round trip changed the rules:\n got %s\nwant %s", got, want)
	}
}

func TestFlattenCorsKeepsNullHeaders(t *testing.T) {
	prior := []corsRuleModel{{AllowedHeaders: types.ListNull(types.StringType), ExposeHeaders: stringsValue([]string{})}}
	rules := []client.CorsRule{{AllowedOrigin: []string{"*"}, AllowedMethod: []string{"GET"}, AllowedHeader: []string{}, ExposeHeader: []string{}}}
	flat := flattenCorsRules(prior, rules)
	if !flat[0].AllowedHeaders.IsNull() {
		t.Fatal("allowed_headers should stay null when Garage returns an empty list")
	}
	if flat[0].ExposeHeaders.IsNull() || len(flat[0].ExposeHeaders.Elements()) != 0 {
		t.Fatal("expose_headers should stay an empty list when it was configured as one")
	}
}

func TestUpdateRequestManagesRulesOnlyWhenSet(t *testing.T) {
	m := bucketModel{MaxSize: types.Int64Null(), MaxObjects: types.Int64Value(5)}
	req := m.updateRequest(nil)
	if req.CorsRules != nil || req.LifecycleRules != nil {
		t.Fatal("rules must not be sent when they are not configured")
	}
	if got := mustJSON(t, req.Quotas); got != `{"maxSize":null,"maxObjects":5}` {
		t.Fatalf("quotas = %s", got)
	}
	if got := mustJSON(t, req.WebsiteAccess); got != `{"enabled":false}` {
		t.Fatalf("website = %s", got)
	}

	prior := bucketModel{CorsRules: []corsRuleModel{{}}}
	req = m.updateRequest(&prior)
	if req.CorsRules == nil || len(*req.CorsRules) != 0 || mustJSON(t, req.CorsRules) != `[]` {
		t.Fatal("removing cors_rules from the configuration must clear them once")
	}
}

func TestSelectGlobalAlias(t *testing.T) {
	aliases := []string{"a", "b"}
	if got := selectGlobalAlias(types.StringValue("b"), aliases, false); got.ValueString() != "b" {
		t.Fatalf("got %s", got)
	}
	if got := selectGlobalAlias(types.StringValue("c"), aliases, false); !got.IsNull() {
		t.Fatalf("a removed alias must read as null, got %s", got)
	}
	if got := selectGlobalAlias(types.StringNull(), aliases, true); got.ValueString() != "a" {
		t.Fatalf("an import must pick the first alias, got %s", got)
	}
	if got := selectGlobalAlias(types.StringNull(), aliases, false); !got.IsNull() {
		t.Fatalf("an unmanaged alias must stay null, got %s", got)
	}
}

func TestLayoutChanges(t *testing.T) {
	capacity := int64(1000)
	two := int64(2)
	layout := &client.ClusterLayout{
		Version: 3,
		Roles: []client.LayoutRole{
			{ID: "same", Zone: "z1", Capacity: &capacity, Tags: []string{"b", "a"}},
			{ID: "changed", Zone: "z1", Capacity: &capacity},
			{ID: "gone", Zone: "z2", Capacity: &capacity},
		},
	}
	m := &clusterLayoutModel{
		ZoneRedundancy: types.Int64Value(two),
		Roles: map[string]layoutRoleModel{
			"same":    {Zone: types.StringValue("z1"), Capacity: types.Int64Value(1000), Tags: setOf("a", "b")},
			"changed": {Zone: types.StringValue("z3"), Capacity: types.Int64Value(1000), Tags: types.SetNull(types.StringType)},
			"new":     {Zone: types.StringValue("z2"), Capacity: types.Int64Null(), Tags: types.SetNull(types.StringType)},
		},
	}
	got := mustJSON(t, layoutChanges(layout, m))
	want := `{"parameters":{"zoneRedundancy":{"atLeast":2}},"roles":[` +
		`{"capacity":1000,"id":"changed","tags":[],"zone":"z3"},` +
		`{"capacity":null,"id":"new","tags":[],"zone":"z2"},` +
		`{"id":"gone","remove":true}]}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func setOf(values ...string) types.Set {
	set, _ := types.SetValueFrom(t0, types.StringType, values)
	return set
}
