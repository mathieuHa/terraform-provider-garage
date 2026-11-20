package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var _ provider.Provider = (*garageProvider)(nil)

type garageProvider struct {
	version string
}

type providerModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	Token      types.String `tfsdk:"token"`
	CACertFile types.String `tfsdk:"ca_cert_file"`
	Insecure   types.Bool   `tfsdk:"insecure"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &garageProvider{version: version}
	}
}

func (p *garageProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "garage"
	resp.Version = p.version
}

func (p *garageProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages [Garage](https://garagehq.deuxfleurs.fr/) through its admin API v2: buckets, aliases, access keys, permissions, admin tokens and the cluster layout.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URL of the Garage admin API, for example `http://127.0.0.1:3903`. Can also be set with `GARAGE_ENDPOINT`.",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Admin API bearer token: the `admin_token` of the daemon configuration, or a token created with `garage_admin_token`. Can also be set with `GARAGE_TOKEN`.",
			},
			"ca_cert_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to a PEM file with the CA certificates that sign the admin API certificate, added to the system pool. Can also be set with `GARAGE_CA_CERT_FILE`.",
			},
			"insecure": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Only for tests.",
			},
		},
	}
}

func (p *garageProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for attr, value := range map[string]types.String{"endpoint": config.Endpoint, "token": config.Token, "ca_cert_file": config.CACertFile} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Unknown Garage provider setting",
				fmt.Sprintf("The provider cannot be configured while %q is unknown. Set it to a value known at plan time.", attr))
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := stringOrEnv(config.Endpoint, "GARAGE_ENDPOINT")
	token := stringOrEnv(config.Token, "GARAGE_TOKEN")
	caFile := stringOrEnv(config.CACertFile, "GARAGE_CA_CERT_FILE")

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing Garage admin API endpoint",
			"Set the endpoint attribute or the GARAGE_ENDPOINT environment variable.")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing Garage admin API token",
			"Set the token attribute or the GARAGE_TOKEN environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ca_cert_file"), "Cannot read the CA file", err.Error())
			return
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			resp.Diagnostics.AddAttributeError(path.Root("ca_cert_file"), "Invalid CA file", "The file contains no PEM certificate.")
			return
		}
		tlsConfig.RootCAs = pool
	}
	if config.Insecure.ValueBool() {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	c := client.New(endpoint, token, "terraform-provider-garage/"+p.version, &http.Client{Transport: transport})
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *garageProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newBucketResource,
		newBucketAliasResource,
		newBucketPermissionResource,
		newKeyResource,
		newAdminTokenResource,
		newClusterLayoutResource,
	}
}

func (p *garageProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newBucketDataSource,
		newBucketsDataSource,
		newKeyDataSource,
		newKeysDataSource,
		newAdminTokenDataSource,
		newClusterHealthDataSource,
		newClusterStatusDataSource,
		newClusterLayoutDataSource,
		newNodeDataSource,
	}
}

func stringOrEnv(v types.String, env string) string {
	if !v.IsNull() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}
