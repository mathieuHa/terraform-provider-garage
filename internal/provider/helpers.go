package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

func clientFromResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return nil
	}
	return c
}

func clientFromDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return nil
	}
	return c
}

func stringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

func stringValue(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func int64Value(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func stringsValue(values []string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func optionalStringsValue(prior types.List, values []string) types.List {
	if len(values) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType)
	}
	return stringsValue(values)
}

func stringSlice(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	out := make([]string, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		if s, ok := e.(types.String); ok {
			out = append(out, s.ValueString())
		}
	}
	return out
}

func rfc3339Value(p *string) timetypes.RFC3339 {
	if p == nil || *p == "" {
		return timetypes.NewRFC3339Null()
	}
	t, err := time.Parse(time.RFC3339Nano, *p)
	if err != nil {
		return timetypes.NewRFC3339ValueMust(*p)
	}
	return timetypes.NewRFC3339TimeValue(t)
}

func keepEquivalentTime(prior timetypes.RFC3339, read timetypes.RFC3339) timetypes.RFC3339 {
	if prior.IsNull() || prior.IsUnknown() || read.IsNull() {
		return read
	}
	pt, d1 := prior.ValueRFC3339Time()
	rt, d2 := read.ValueRFC3339Time()
	if d1.HasError() || d2.HasError() {
		return read
	}
	if pt.Equal(rt) {
		return prior
	}
	return read
}

func rfc3339String(v timetypes.RFC3339) (*string, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	t, diags := v.ValueRFC3339Time()
	if diags.HasError() {
		return nil, diags
	}
	s := t.UTC().Format(time.RFC3339)
	return &s, nil
}

var (
	keyIDRegex     = regexp.MustCompile(`^GK[0-9a-f]{24}$`)
	keySecretRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type stateSetter interface {
	Set(ctx context.Context, val any) diag.Diagnostics
}

type diagAppender interface {
	AddError(summary, detail string)
	Append(in ...diag.Diagnostic)
}
