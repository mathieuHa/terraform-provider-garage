package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var (
	_ resource.Resource                = (*bucketResource)(nil)
	_ resource.ResourceWithImportState = (*bucketResource)(nil)
)

var (
	bucketIDRegex   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
)

type bucketResource struct {
	client *client.Client
}

func newBucketResource() resource.Resource {
	return &bucketResource{}
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromResource(req, resp)
}

func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Garage bucket with its quotas, website hosting, CORS rules and lifecycle rules.\n\n" +
			"`website` and the quotas are authoritative: removing them from the configuration disables website hosting and removes the quotas. " +
			"`cors_rules` and `lifecycle_rules` are authoritative only when set: when they are absent, rules set through the S3 API are left alone, and setting them to `[]` removes every rule.\n\n" +
			"Garage refuses to delete a bucket that still contains objects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Bucket ID (64 hexadecimal characters).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"global_alias": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Global alias of the bucket, which is its name in the S3 API. A change adds the new alias before it removes the old one. Use `garage_bucket_alias` for additional global aliases and for local aliases.",
				Validators:          []validator.String{stringvalidator.RegexMatches(bucketNameRegex, "must be a valid S3 bucket name")},
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation date of the bucket.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"max_size": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum total size of the objects in the bucket, in bytes. Unset means no quota.",
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"max_objects": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of objects in the bucket. Unset means no quota.",
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"website":         websiteSchema(),
			"cors_rules":      corsRulesSchema(),
			"lifecycle_rules": lifecycleRulesSchema(),
		},
	}
}

func websiteSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Serve the bucket as a static website on the Garage web endpoint. Absent means website hosting is disabled.",
		Attributes: map[string]schema.Attribute{
			"index_document": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Document served for a directory, for example `index.html`.",
			},
			"error_document": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Document served on errors, for example `error.html`.",
			},
			"routing_rules": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Redirection rules, evaluated in order.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"condition": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "When the rule applies. Absent means always.",
							Attributes: map[string]schema.Attribute{
								"key_prefix_equals": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "Object key prefix that the request must match.",
								},
								"http_error_code_returned_equals": schema.Int64Attribute{
									Optional:            true,
									MarkdownDescription: "HTTP error code that the request must produce, for example `404`.",
									Validators:          []validator.Int64{int64validator.Between(400, 599)},
								},
							},
						},
						"redirect": schema.SingleNestedAttribute{
							Required:            true,
							MarkdownDescription: "Where to redirect.",
							Attributes: map[string]schema.Attribute{
								"host_name": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "Host name of the redirection.",
								},
								"protocol": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "`http` or `https`.",
									Validators:          []validator.String{stringvalidator.OneOf("http", "https")},
								},
								"http_redirect_code": schema.Int64Attribute{
									Optional:            true,
									Computed:            true,
									Default:             int64default.StaticInt64(302),
									MarkdownDescription: "HTTP status code of the redirection. Defaults to `302`.",
									Validators:          []validator.Int64{int64validator.Between(200, 399)},
								},
								"replace_key_prefix_with": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "Replaces the prefix matched by `key_prefix_equals`.",
									Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("replace_key_with"))},
								},
								"replace_key_with": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "Replaces the whole key.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func corsRulesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:            true,
		MarkdownDescription: "CORS rules of the bucket. Absent means the rules are not managed by this resource; `[]` removes every rule.",
		Validators:          []validator.List{listvalidator.SizeAtMost(100)},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Identifier of the rule.",
				},
				"allowed_origins": schema.ListAttribute{
					Required:            true,
					ElementType:         types.StringType,
					MarkdownDescription: "Origins allowed to make cross-origin requests, for example `https://app.example.com` or `*`.",
					Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				},
				"allowed_methods": schema.ListAttribute{
					Required:            true,
					ElementType:         types.StringType,
					MarkdownDescription: "HTTP methods allowed: `GET`, `PUT`, `POST`, `DELETE`, `HEAD`.",
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
						listvalidator.ValueStringsAre(stringvalidator.OneOf("GET", "PUT", "POST", "DELETE", "HEAD")),
					},
				},
				"allowed_headers": schema.ListAttribute{
					Optional:            true,
					ElementType:         types.StringType,
					MarkdownDescription: "Request headers allowed in a preflight request.",
				},
				"expose_headers": schema.ListAttribute{
					Optional:            true,
					ElementType:         types.StringType,
					MarkdownDescription: "Response headers that browsers can read.",
				},
				"max_age_seconds": schema.Int64Attribute{
					Optional:            true,
					MarkdownDescription: "How long browsers can cache a preflight response.",
					Validators:          []validator.Int64{int64validator.AtLeast(0)},
				},
			},
		},
	}
}

func lifecycleRulesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Lifecycle rules of the bucket. Absent means the rules are not managed by this resource; `[]` removes every rule.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Identifier of the rule.",
				},
				"enabled": schema.BoolAttribute{
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(true),
					MarkdownDescription: "Whether the rule applies. Defaults to `true`.",
				},
				"filter": schema.SingleNestedAttribute{
					Optional:            true,
					MarkdownDescription: "Objects the rule applies to. All conditions must match. Absent means every object.",
					Attributes: map[string]schema.Attribute{
						"prefix": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Object key prefix.",
						},
						"object_size_greater_than": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Minimum object size, in bytes (exclusive).",
							Validators:          []validator.Int64{int64validator.AtLeast(0)},
						},
						"object_size_less_than": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Maximum object size, in bytes (exclusive).",
							Validators:          []validator.Int64{int64validator.AtLeast(1)},
						},
					},
				},
				"expiration_days": schema.Int64Attribute{
					Optional:            true,
					MarkdownDescription: "Delete objects this many days after their creation.",
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
						int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("expiration_date")),
					},
				},
				"expiration_date": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Delete objects from this date on, at midnight UTC, for example `2027-01-01T00:00:00Z`.",
				},
				"abort_incomplete_multipart_upload_days": schema.Int64Attribute{
					Optional:            true,
					MarkdownDescription: "Abort multipart uploads that are not completed this many days after they start.",
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
			},
			Validators: []validator.Object{
				objectvalidator.AtLeastOneOf(
					path.MatchRelative().AtName("expiration_days"),
					path.MatchRelative().AtName("expiration_date"),
					path.MatchRelative().AtName("abort_incomplete_multipart_upload_days"),
				),
			},
		},
	}
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateBucket(ctx, client.CreateBucketRequest{GlobalAlias: stringPtr(plan.GlobalAlias)})
	if err != nil {
		resp.Diagnostics.AddError("Cannot create the bucket", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), created.ID)...)

	bucket, err := r.client.UpdateBucket(ctx, created.ID, plan.updateRequest(nil))
	if err != nil {
		resp.Diagnostics.AddError("Cannot configure the bucket",
			fmt.Sprintf("The bucket %s was created but its configuration failed, and it is now tainted: %s", created.ID, err))
		return
	}

	plan.readFrom(bucket, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, err := r.client.GetBucket(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the bucket", err.Error())
		return
	}

	state.readFrom(bucket, state.Created.IsNull())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	if !plan.GlobalAlias.Equal(state.GlobalAlias) {
		if !plan.GlobalAlias.IsNull() {
			if _, err := r.client.AddBucketAlias(ctx, id, plan.GlobalAlias.ValueString(), ""); err != nil {
				resp.Diagnostics.AddError("Cannot add the global alias", err.Error())
				return
			}
		}
		if !state.GlobalAlias.IsNull() {
			if _, err := r.client.RemoveBucketAlias(ctx, id, state.GlobalAlias.ValueString(), ""); err != nil {
				resp.Diagnostics.AddError("Cannot remove the previous global alias",
					fmt.Sprintf("%s\n\nGarage refuses to remove the last alias of a bucket: give it another alias first.", err))
				return
			}
		}
	}

	bucket, err := r.client.UpdateBucket(ctx, id, plan.updateRequest(&state))
	if err != nil {
		resp.Diagnostics.AddError("Cannot update the bucket", err.Error())
		return
	}

	plan.readFrom(bucket, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBucket(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Cannot delete the bucket", err.Error())
	}
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if !bucketIDRegex.MatchString(id) {
		bucket, err := r.client.GetBucketByGlobalAlias(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Cannot find the bucket", fmt.Sprintf("No bucket has the ID or global alias %q: %s", id, err))
			return
		}
		id = bucket.ID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("global_alias"), req.ID)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
