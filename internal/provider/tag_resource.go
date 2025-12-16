// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/resource_tag"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TagResource{}
var _ resource.ResourceWithImportState = &TagResource{}

func NewTagResource() resource.Resource {
	return &TagResource{}
}

// TagResource defines the resource implementation.
type TagResource struct {
	client *LakeFSClient
}

// TagCreateRequest represents the request to create a tag
type TagCreateRequest struct {
	ID  string `json:"id"`
	Ref string `json:"ref"`
}

// TagResponse represents the API response for a tag
type TagResponse struct {
	ID       string `json:"id"`
	CommitID string `json:"commit_id"`
}

func (r *TagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_tag.TagResourceSchema(ctx)
}

func (r *TagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*LakeFSClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *LakeFSClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_tag.TagModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	// In the generated schema, Id is the tag name (required field)
	tagName := data.Id.ValueString()

	createReq := TagCreateRequest{
		ID:  tagName,
		Ref: data.Ref.ValueString(),
	}

	tflog.Debug(ctx, "Creating tag", map[string]any{
		"repository": repository,
		"tag":        createReq.ID,
		"ref":        createReq.Ref,
	})

	var result TagResponse
	err := client.Post(ctx, fmt.Sprintf("/repositories/%s/tags", repository), createReq, &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tag: %s", err))
		return
	}

	// Map response to state - keep Id as tag name per the schema
	data.CommitId = types.StringValue(result.CommitID)
	data.Tag = types.StringValue(tagName)

	tflog.Trace(ctx, "Created tag", map[string]any{
		"id":        data.Id.ValueString(),
		"commit_id": result.CommitID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_tag.TagModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	// Id is the tag name in the generated schema
	tagName := data.Id.ValueString()
	if tagName == "" {
		tagName = data.Tag.ValueString()
	}

	var result TagResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/tags/%s", repository, tagName), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tag: %s", err))
		return
	}

	// Map response to state
	data.CommitId = types.StringValue(result.CommitID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_tag.TagModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Tags in LakeFS are immutable
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_tag.TagModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	// Id is the tag name in the generated schema
	tagName := data.Id.ValueString()
	if tagName == "" {
		tagName = data.Tag.ValueString()
	}

	tflog.Debug(ctx, "Deleting tag", map[string]any{
		"repository": repository,
		"tag":        tagName,
	})

	err := client.Delete(ctx, fmt.Sprintf("/repositories/%s/tags/%s", repository, tagName))
	if err != nil {
		if !IsNotFound(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tag: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "Deleted tag", map[string]any{
		"repository": repository,
		"tag":        tagName,
	})
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: repository/tag
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'repository/tag', got: %s", req.ID),
		)
		return
	}

	client := NewAPIClient(r.client)
	repository := parts[0]
	tagName := parts[1]

	var result TagResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/tags/%s", repository, tagName), &result)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to import tag %s: %s", req.ID, err))
		return
	}

	var data resource_tag.TagModel
	data.Id = types.StringValue(tagName) // Id is the tag name
	data.Repository = types.StringValue(repository)
	data.Tag = types.StringValue(tagName)
	data.CommitId = types.StringValue(result.CommitID)
	data.Ref = types.StringValue(result.CommitID) // Use commit ID as ref
	data.Force = types.BoolValue(false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
