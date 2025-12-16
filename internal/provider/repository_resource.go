// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/resource_repository"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}

func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	client *LakeFSClient
}

// RepositoryCreateRequest represents the request to create a repository
type RepositoryCreateRequest struct {
	Name             string `json:"name"`
	StorageNamespace string `json:"storage_namespace"`
	DefaultBranch    string `json:"default_branch,omitempty"`
	SampleData       bool   `json:"sample_data,omitempty"`
	ReadOnly         bool   `json:"read_only,omitempty"`
}

// RepositoryResponse represents the API response for a repository
type RepositoryResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name,omitempty"`
	StorageNamespace string `json:"storage_namespace"`
	StorageID        string `json:"storage_id,omitempty"`
	DefaultBranch    string `json:"default_branch"`
	CreationDate     int64  `json:"creation_date"`
	ReadOnly         bool   `json:"read_only,omitempty"`
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_repository.RepositoryResourceSchema(ctx)
}

func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_repository.RepositoryModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	createReq := RepositoryCreateRequest{
		Name:             data.Name.ValueString(),
		StorageNamespace: data.StorageNamespace.ValueString(),
	}

	if !data.DefaultBranch.IsNull() && !data.DefaultBranch.IsUnknown() {
		createReq.DefaultBranch = data.DefaultBranch.ValueString()
	}

	if !data.SampleData.IsNull() && !data.SampleData.IsUnknown() {
		createReq.SampleData = data.SampleData.ValueBool()
	}

	if !data.ReadOnly.IsNull() && !data.ReadOnly.IsUnknown() {
		createReq.ReadOnly = data.ReadOnly.ValueBool()
	}

	tflog.Debug(ctx, "Creating repository", map[string]any{
		"name":              createReq.Name,
		"storage_namespace": createReq.StorageNamespace,
	})

	var result RepositoryResponse
	err := client.Post(ctx, "/repositories", createReq, &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create repository: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(result.ID)
	data.Repository = types.StringValue(result.ID)
	data.Name = types.StringValue(result.ID) // LakeFS uses ID as name
	data.StorageNamespace = types.StringValue(result.StorageNamespace)
	data.StorageId = types.StringValue(result.StorageID)
	data.DefaultBranch = types.StringValue(result.DefaultBranch)
	data.CreationDate = types.Int64Value(result.CreationDate)
	data.ReadOnly = types.BoolValue(result.ReadOnly)

	tflog.Trace(ctx, "Created repository", map[string]any{"id": result.ID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_repository.RepositoryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repoID := data.Id.ValueString()
	if repoID == "" {
		repoID = data.Name.ValueString()
	}

	var result RepositoryResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s", repoID), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repository: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(result.ID)
	data.Repository = types.StringValue(result.ID)
	data.Name = types.StringValue(result.ID)
	data.StorageNamespace = types.StringValue(result.StorageNamespace)
	data.StorageId = types.StringValue(result.StorageID)
	data.DefaultBranch = types.StringValue(result.DefaultBranch)
	data.CreationDate = types.Int64Value(result.CreationDate)
	data.ReadOnly = types.BoolValue(result.ReadOnly)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_repository.RepositoryModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// LakeFS repositories are immutable - most attributes cannot be updated
	// We just save the state as-is
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_repository.RepositoryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repoID := data.Id.ValueString()
	if repoID == "" {
		repoID = data.Name.ValueString()
	}

	tflog.Debug(ctx, "Deleting repository", map[string]any{"id": repoID})

	err := client.Delete(ctx, fmt.Sprintf("/repositories/%s", repoID))
	if err != nil {
		if !IsNotFound(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete repository: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "Deleted repository", map[string]any{"id": repoID})
}

func (r *RepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client := NewAPIClient(r.client)

	var result RepositoryResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s", req.ID), &result)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to import repository %s: %s", req.ID, err))
		return
	}

	var data resource_repository.RepositoryModel
	data.Id = types.StringValue(result.ID)
	data.Repository = types.StringValue(result.ID)
	data.Name = types.StringValue(result.ID)
	data.StorageNamespace = types.StringValue(result.StorageNamespace)
	data.StorageId = types.StringValue(result.StorageID)
	data.DefaultBranch = types.StringValue(result.DefaultBranch)
	data.CreationDate = types.Int64Value(result.CreationDate)
	data.ReadOnly = types.BoolValue(result.ReadOnly)
	data.SampleData = types.BoolValue(false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
