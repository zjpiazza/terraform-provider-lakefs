// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/datasource_commit"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CommitDataSource{}

func NewCommitDataSource() datasource.DataSource {
	return &CommitDataSource{}
}

// CommitDataSource defines the data source implementation.
type CommitDataSource struct {
	client *LakeFSClient
}

// CommitResponse represents the API response for a commit
type CommitResponse struct {
	ID           string            `json:"id"`
	Committer    string            `json:"committer"`
	Message      string            `json:"message"`
	MetaRangeID  string            `json:"meta_range_id"`
	CreationDate int64             `json:"creation_date"`
	Parents      []string          `json:"parents"`
	Metadata     map[string]string `json:"metadata"`
	Generation   int64             `json:"generation"`
	Version      int64             `json:"version"`
}

func (d *CommitDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_commit"
}

func (d *CommitDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_commit.CommitDataSourceSchema(ctx)
}

func (d *CommitDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CommitDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_commit.CommitModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(d.client)

	repository := data.Repository.ValueString()
	commitID := data.CommitId.ValueString()

	var result CommitResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/commits/%s", repository, commitID), &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read commit: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(result.ID)
	data.Committer = types.StringValue(result.Committer)
	data.Message = types.StringValue(result.Message)
	data.MetaRangeId = types.StringValue(result.MetaRangeID)
	data.CreationDate = types.Int64Value(result.CreationDate)
	data.Generation = types.Int64Value(result.Generation)
	data.Version = types.Int64Value(result.Version)

	// Convert parents to list
	if len(result.Parents) > 0 {
		var parentValues []attr.Value
		for _, p := range result.Parents {
			parentValues = append(parentValues, types.StringValue(p))
		}
		parentsList, _ := types.ListValue(types.StringType, parentValues)
		data.Parents = parentsList
	} else {
		data.Parents = types.ListNull(types.StringType)
	}

	// Convert metadata to map
	if len(result.Metadata) > 0 {
		metadataValues := make(map[string]attr.Value)
		for k, v := range result.Metadata {
			metadataValues[k] = types.StringValue(v)
		}
		metadataMap, _ := types.MapValue(types.StringType, metadataValues)
		data.Metadata = metadataMap
	} else {
		data.Metadata = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
