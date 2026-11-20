package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

type bucketModel struct {
	ID             types.String         `tfsdk:"id"`
	GlobalAlias    types.String         `tfsdk:"global_alias"`
	Created        types.String         `tfsdk:"created"`
	MaxSize        types.Int64          `tfsdk:"max_size"`
	MaxObjects     types.Int64          `tfsdk:"max_objects"`
	Website        *websiteModel        `tfsdk:"website"`
	CorsRules      []corsRuleModel      `tfsdk:"cors_rules"`
	LifecycleRules []lifecycleRuleModel `tfsdk:"lifecycle_rules"`
}

type websiteModel struct {
	IndexDocument types.String       `tfsdk:"index_document"`
	ErrorDocument types.String       `tfsdk:"error_document"`
	RoutingRules  []routingRuleModel `tfsdk:"routing_rules"`
}

type routingRuleModel struct {
	Condition *routingConditionModel `tfsdk:"condition"`
	Redirect  routingRedirectModel   `tfsdk:"redirect"`
}

type routingConditionModel struct {
	KeyPrefixEquals             types.String `tfsdk:"key_prefix_equals"`
	HTTPErrorCodeReturnedEquals types.Int64  `tfsdk:"http_error_code_returned_equals"`
}

type routingRedirectModel struct {
	HostName             types.String `tfsdk:"host_name"`
	Protocol             types.String `tfsdk:"protocol"`
	HTTPRedirectCode     types.Int64  `tfsdk:"http_redirect_code"`
	ReplaceKeyPrefixWith types.String `tfsdk:"replace_key_prefix_with"`
	ReplaceKeyWith       types.String `tfsdk:"replace_key_with"`
}

type corsRuleModel struct {
	ID             types.String `tfsdk:"id"`
	AllowedOrigins types.List   `tfsdk:"allowed_origins"`
	AllowedMethods types.List   `tfsdk:"allowed_methods"`
	AllowedHeaders types.List   `tfsdk:"allowed_headers"`
	ExposeHeaders  types.List   `tfsdk:"expose_headers"`
	MaxAgeSeconds  types.Int64  `tfsdk:"max_age_seconds"`
}

type lifecycleRuleModel struct {
	ID                                 types.String          `tfsdk:"id"`
	Enabled                            types.Bool            `tfsdk:"enabled"`
	Filter                             *lifecycleFilterModel `tfsdk:"filter"`
	ExpirationDays                     types.Int64           `tfsdk:"expiration_days"`
	ExpirationDate                     types.String          `tfsdk:"expiration_date"`
	AbortIncompleteMultipartUploadDays types.Int64           `tfsdk:"abort_incomplete_multipart_upload_days"`
}

type lifecycleFilterModel struct {
	Prefix                types.String `tfsdk:"prefix"`
	ObjectSizeGreaterThan types.Int64  `tfsdk:"object_size_greater_than"`
	ObjectSizeLessThan    types.Int64  `tfsdk:"object_size_less_than"`
}

func (m *bucketModel) updateRequest(prior *bucketModel) client.UpdateBucketRequest {
	req := client.UpdateBucketRequest{
		Quotas:        &client.Quotas{MaxSize: int64Ptr(m.MaxSize), MaxObjects: int64Ptr(m.MaxObjects)},
		WebsiteAccess: expandWebsite(m.Website),
	}
	if m.CorsRules != nil || (prior != nil && prior.CorsRules != nil) {
		rules := expandCorsRules(m.CorsRules)
		req.CorsRules = &rules
	}
	if m.LifecycleRules != nil || (prior != nil && prior.LifecycleRules != nil) {
		rules := expandLifecycleRules(m.LifecycleRules)
		req.LifecycleRules = &rules
	}
	return req
}

func (m *bucketModel) readFrom(b *client.Bucket, importing bool) {
	m.ID = types.StringValue(b.ID)
	m.Created = types.StringValue(b.Created)
	m.GlobalAlias = selectGlobalAlias(m.GlobalAlias, b.GlobalAliases, importing)
	m.MaxSize = int64Value(b.Quotas.MaxSize)
	m.MaxObjects = int64Value(b.Quotas.MaxObjects)
	m.Website = flattenWebsite(m.Website, b)
	if m.CorsRules != nil || (importing && len(b.CorsRules) > 0) {
		m.CorsRules = flattenCorsRules(m.CorsRules, b.CorsRules)
	}
	if m.LifecycleRules != nil || (importing && len(b.LifecycleRules) > 0) {
		m.LifecycleRules = flattenLifecycleRules(m.LifecycleRules, b.LifecycleRules)
	}
}

func selectGlobalAlias(prior types.String, aliases []string, importing bool) types.String {
	if !prior.IsNull() && !prior.IsUnknown() {
		for _, a := range aliases {
			if a == prior.ValueString() {
				return prior
			}
		}
		return types.StringNull()
	}
	if importing && len(aliases) > 0 {
		return types.StringValue(aliases[0])
	}
	return types.StringNull()
}

func expandWebsite(w *websiteModel) *client.WebsiteAccessUpdate {
	if w == nil {
		return &client.WebsiteAccessUpdate{Enabled: false}
	}
	update := &client.WebsiteAccessUpdate{
		Enabled:       true,
		IndexDocument: stringPtr(w.IndexDocument),
		ErrorDocument: stringPtr(w.ErrorDocument),
		RoutingRules:  []client.RoutingRule{},
	}
	for _, r := range w.RoutingRules {
		rule := client.RoutingRule{
			Redirect: client.RoutingRedirect{
				HostName:             stringPtr(r.Redirect.HostName),
				Protocol:             stringPtr(r.Redirect.Protocol),
				HTTPRedirectCode:     int64Ptr(r.Redirect.HTTPRedirectCode),
				ReplaceKeyPrefixWith: stringPtr(r.Redirect.ReplaceKeyPrefixWith),
				ReplaceKeyWith:       stringPtr(r.Redirect.ReplaceKeyWith),
			},
		}
		if r.Condition != nil {
			rule.Condition = &client.RoutingCondition{
				KeyPrefixEquals:             stringPtr(r.Condition.KeyPrefixEquals),
				HTTPErrorCodeReturnedEquals: int64Ptr(r.Condition.HTTPErrorCodeReturnedEquals),
			}
		}
		update.RoutingRules = append(update.RoutingRules, rule)
	}
	return update
}

func flattenWebsite(prior *websiteModel, b *client.Bucket) *websiteModel {
	if !b.WebsiteAccess || b.WebsiteConfig == nil {
		return nil
	}
	cfg := b.WebsiteConfig
	w := &websiteModel{
		IndexDocument: types.StringValue(cfg.IndexDocument),
		ErrorDocument: stringValue(cfg.ErrorDocument),
	}
	if len(cfg.RoutingRules) > 0 || (prior != nil && prior.RoutingRules != nil) {
		w.RoutingRules = make([]routingRuleModel, 0, len(cfg.RoutingRules))
	}
	for i, r := range cfg.RoutingRules {
		var priorRule *routingRuleModel
		if prior != nil && i < len(prior.RoutingRules) {
			priorRule = &prior.RoutingRules[i]
		}
		rule := routingRuleModel{
			Redirect: routingRedirectModel{
				HostName:             stringValue(r.Redirect.HostName),
				Protocol:             stringValue(r.Redirect.Protocol),
				HTTPRedirectCode:     int64Value(r.Redirect.HTTPRedirectCode),
				ReplaceKeyPrefixWith: stringValue(r.Redirect.ReplaceKeyPrefixWith),
				ReplaceKeyWith:       stringValue(r.Redirect.ReplaceKeyWith),
			},
		}
		if c := r.Condition; c != nil && (c.KeyPrefixEquals != nil || c.HTTPErrorCodeReturnedEquals != nil) {
			rule.Condition = &routingConditionModel{
				KeyPrefixEquals:             stringValue(c.KeyPrefixEquals),
				HTTPErrorCodeReturnedEquals: int64Value(c.HTTPErrorCodeReturnedEquals),
			}
		} else if priorRule != nil && priorRule.Condition != nil && priorRule.Condition.KeyPrefixEquals.IsNull() && priorRule.Condition.HTTPErrorCodeReturnedEquals.IsNull() {
			rule.Condition = priorRule.Condition
		}
		w.RoutingRules = append(w.RoutingRules, rule)
	}
	return w
}

func expandCorsRules(rules []corsRuleModel) []client.CorsRule {
	out := make([]client.CorsRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, client.CorsRule{
			ID:            stringPtr(r.ID),
			AllowedOrigin: stringSlice(r.AllowedOrigins),
			AllowedMethod: stringSlice(r.AllowedMethods),
			AllowedHeader: stringSlice(r.AllowedHeaders),
			ExposeHeader:  stringSlice(r.ExposeHeaders),
			MaxAgeSeconds: int64Ptr(r.MaxAgeSeconds),
		})
	}
	return out
}

func flattenCorsRules(prior []corsRuleModel, rules []client.CorsRule) []corsRuleModel {
	out := make([]corsRuleModel, 0, len(rules))
	for i, r := range rules {
		p := corsRuleModel{
			AllowedHeaders: types.ListNull(types.StringType),
			ExposeHeaders:  types.ListNull(types.StringType),
		}
		if i < len(prior) {
			p = prior[i]
		}
		out = append(out, corsRuleModel{
			ID:             stringValue(r.ID),
			AllowedOrigins: stringsValue(r.AllowedOrigin),
			AllowedMethods: stringsValue(r.AllowedMethod),
			AllowedHeaders: optionalStringsValue(p.AllowedHeaders, r.AllowedHeader),
			ExposeHeaders:  optionalStringsValue(p.ExposeHeaders, r.ExposeHeader),
			MaxAgeSeconds:  int64Value(r.MaxAgeSeconds),
		})
	}
	return out
}

func expandLifecycleRules(rules []lifecycleRuleModel) []client.LifecycleRule {
	out := make([]client.LifecycleRule, 0, len(rules))
	for _, r := range rules {
		status := "Enabled"
		if !r.Enabled.IsNull() && !r.Enabled.IsUnknown() && !r.Enabled.ValueBool() {
			status = "Disabled"
		}
		rule := client.LifecycleRule{
			ID:     stringPtr(r.ID),
			Status: status,
			Filter: expandLifecycleFilter(r.Filter),
		}
		if !r.ExpirationDays.IsNull() || !r.ExpirationDate.IsNull() {
			rule.Expiration = &client.LifecycleExpiration{
				Days: int64Ptr(r.ExpirationDays),
				Date: stringPtr(r.ExpirationDate),
			}
		}
		if !r.AbortIncompleteMultipartUploadDays.IsNull() {
			rule.AbortIncompleteMultipartUpload = &client.LifecycleAbortIncompleteUpload{
				DaysAfterInitiation: r.AbortIncompleteMultipartUploadDays.ValueInt64(),
			}
		}
		out = append(out, rule)
	}
	return out
}

func expandLifecycleFilter(f *lifecycleFilterModel) *client.LifecycleFilter {
	if f == nil {
		return nil
	}
	filter := &client.LifecycleFilter{
		Prefix:                stringPtr(f.Prefix),
		ObjectSizeGreaterThan: int64Ptr(f.ObjectSizeGreaterThan),
		ObjectSizeLessThan:    int64Ptr(f.ObjectSizeLessThan),
	}
	conditions := 0
	for _, set := range []bool{filter.Prefix != nil, filter.ObjectSizeGreaterThan != nil, filter.ObjectSizeLessThan != nil} {
		if set {
			conditions++
		}
	}
	switch conditions {
	case 0:
		return nil
	case 1:
		return filter
	default:
		return &client.LifecycleFilter{And: filter}
	}
}

func flattenLifecycleRules(prior []lifecycleRuleModel, rules []client.LifecycleRule) []lifecycleRuleModel {
	out := make([]lifecycleRuleModel, 0, len(rules))
	for i, r := range rules {
		var p *lifecycleRuleModel
		if i < len(prior) {
			p = &prior[i]
		}
		rule := lifecycleRuleModel{
			ID:                                 stringValue(r.ID),
			Enabled:                            types.BoolValue(r.Status != "Disabled"),
			ExpirationDays:                     types.Int64Null(),
			ExpirationDate:                     types.StringNull(),
			AbortIncompleteMultipartUploadDays: types.Int64Null(),
		}
		if r.Expiration != nil {
			rule.ExpirationDays = int64Value(r.Expiration.Days)
			rule.ExpirationDate = stringValue(r.Expiration.Date)
		}
		if r.AbortIncompleteMultipartUpload != nil {
			rule.AbortIncompleteMultipartUploadDays = types.Int64Value(r.AbortIncompleteMultipartUpload.DaysAfterInitiation)
		}
		rule.Filter = flattenLifecycleFilter(p, r.Filter)
		out = append(out, rule)
	}
	return out
}

func flattenLifecycleFilter(prior *lifecycleRuleModel, f *client.LifecycleFilter) *lifecycleFilterModel {
	if f != nil && f.And != nil {
		f = f.And
	}
	if f == nil || (f.Prefix == nil && f.ObjectSizeGreaterThan == nil && f.ObjectSizeLessThan == nil) {
		if prior != nil && prior.Filter != nil && prior.Filter.Prefix.IsNull() && prior.Filter.ObjectSizeGreaterThan.IsNull() && prior.Filter.ObjectSizeLessThan.IsNull() {
			return prior.Filter
		}
		return nil
	}
	return &lifecycleFilterModel{
		Prefix:                stringValue(f.Prefix),
		ObjectSizeGreaterThan: int64Value(f.ObjectSizeGreaterThan),
		ObjectSizeLessThan:    int64Value(f.ObjectSizeLessThan),
	}
}
