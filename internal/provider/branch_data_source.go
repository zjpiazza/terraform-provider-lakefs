// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/datasource_branch"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BranchDataSource{}

func NewBranchDataSource() datasource.DataSource {
	return &BranchDataSource{}
}

// BranchDataSource defines the data source implementation.
type BranchDataSource struct {
	client *LakeFSClient
}

func (d *BranchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (d *BranchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_branch.BranchDataSourceSchema(ctx)
}

func (d *BranchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_branch.BranchModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(d.client)

	repository := data.Repository.ValueString()
	branch := data.Branch.ValueString()

	var result BranchResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/branches/%s", repository, branch), &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read branch: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(fmt.Sprintf("%s/%s", repository, branch))
	data.CommitId = types.StringValue(result.CommitID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
