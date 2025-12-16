// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure LakeFSProvider satisfies various provider interfaces.
var _ provider.Provider = &LakeFSProvider{}

// LakeFSProvider defines the provider implementation.
type LakeFSProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// LakeFSProviderModel describes the provider data model.
type LakeFSProviderModel struct {
	Endpoint        types.String `tfsdk:"endpoint"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	SkipSSLVerify   types.Bool   `tfsdk:"skip_ssl_verify"`
}

// LakeFSClient holds the configuration for connecting to LakeFS
type LakeFSClient struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SkipSSLVerify   bool
}

func (p *LakeFSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lakefs"
	resp.Version = p.version
}

func (p *LakeFSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing LakeFS resources. LakeFS is an open source data version control system for data lakes.",
		MarkdownDescription: `
The LakeFS provider allows you to manage LakeFS resources using Terraform.

LakeFS is an open source data version control system for data lakes. It provides Git-like operations
such as branching, committing, and merging for your data.

## Example Usage

` + "```hcl" + `
provider "lakefs" {
  endpoint          = "http://localhost:8000/api/v1"
  access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
` + "```" + `

## Authentication

The provider supports authentication via access key credentials. You can configure
credentials in the provider block or via environment variables:

- ` + "`LAKEFS_ENDPOINT`" + ` - The LakeFS server endpoint
- ` + "`LAKEFS_ACCESS_KEY_ID`" + ` - The access key ID
- ` + "`LAKEFS_SECRET_ACCESS_KEY`" + ` - The secret access key
`,
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The LakeFS server endpoint URL (e.g., http://localhost:8000/api/v1). Can also be set via LAKEFS_ENDPOINT environment variable.",
				Optional:    true,
			},
			"access_key_id": schema.StringAttribute{
				Description: "The access key ID for LakeFS authentication. Can also be set via LAKEFS_ACCESS_KEY_ID environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret_access_key": schema.StringAttribute{
				Description: "The secret access key for LakeFS authentication. Can also be set via LAKEFS_SECRET_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"skip_ssl_verify": schema.BoolAttribute{
				Description: "Skip SSL certificate verification. Default is false.",
				Optional:    true,
			},
		},
	}
}

func (p *LakeFSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring LakeFS client")

	var config LakeFSProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values from environment variables
	endpoint := os.Getenv("LAKEFS_ENDPOINT")
	accessKeyID := os.Getenv("LAKEFS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("LAKEFS_SECRET_ACCESS_KEY")
	skipSSLVerify := false

	// Override with provider configuration if set
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if !config.AccessKeyID.IsNull() {
		accessKeyID = config.AccessKeyID.ValueString()
	}
	if !config.SecretAccessKey.IsNull() {
		secretAccessKey = config.SecretAccessKey.ValueString()
	}
	if !config.SkipSSLVerify.IsNull() {
		skipSSLVerify = config.SkipSSLVerify.ValueBool()
	}

	// Validate required configuration
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing LakeFS API Endpoint",
			"The provider cannot create the LakeFS API client as there is a missing or empty value for the LakeFS API endpoint. "+
				"Set the endpoint value in the configuration or use the LAKEFS_ENDPOINT environment variable.",
		)
	}

	if accessKeyID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key_id"),
			"Missing LakeFS Access Key ID",
			"The provider cannot create the LakeFS API client as there is a missing or empty value for the LakeFS access key ID. "+
				"Set the access_key_id value in the configuration or use the LAKEFS_ACCESS_KEY_ID environment variable.",
		)
	}

	if secretAccessKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_access_key"),
			"Missing LakeFS Secret Access Key",
			"The provider cannot create the LakeFS API client as there is a missing or empty value for the LakeFS secret access key. "+
				"Set the secret_access_key value in the configuration or use the LAKEFS_SECRET_ACCESS_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create client configuration
	client := &LakeFSClient{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SkipSSLVerify:   skipSSLVerify,
	}

	tflog.Debug(ctx, "Created LakeFS client", map[string]any{
		"endpoint": endpoint,
	})

	// Make the client available to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *LakeFSProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRepositoryResource,
		NewBranchResource,
		NewTagResource,
		NewBranchProtectionResource,
	}
}

func (p *LakeFSProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRepositoryDataSource,
		NewBranchDataSource,
		NewCommitDataSource,
		NewCurrentUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LakeFSProvider{
			version: version,
		}
	}
}
