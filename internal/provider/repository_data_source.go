// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/datasource_repository"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RepositoryDataSource{}

func NewRepositoryDataSource() datasource.DataSource {
	return &RepositoryDataSource{}
}

// RepositoryDataSource defines the data source implementation.
type RepositoryDataSource struct {
	client *LakeFSClient
}

func (d *RepositoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *RepositoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_repository.RepositoryDataSourceSchema(ctx)
}

func (d *RepositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*LakeFSClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *LakeFSClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *RepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_repository.RepositoryModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(d.client)

	repoID := data.Repository.ValueString()

	var result RepositoryResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s", repoID), &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repository: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(result.ID)
	data.StorageNamespace = types.StringValue(result.StorageNamespace)
	data.StorageId = types.StringValue(result.StorageID)
	data.DefaultBranch = types.StringValue(result.DefaultBranch)
	data.CreationDate = types.Int64Value(result.CreationDate)
	data.ReadOnly = types.BoolValue(result.ReadOnly)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
