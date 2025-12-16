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

	"github.com/zjpiazza/terraform-provider-lakefs/internal/provider/resource_branch"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BranchResource{}
var _ resource.ResourceWithImportState = &BranchResource{}

func NewBranchResource() resource.Resource {
	return &BranchResource{}
}

// BranchResource defines the resource implementation.
type BranchResource struct {
	client *LakeFSClient
}

// BranchCreateRequest represents the request to create a branch
type BranchCreateRequest struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// BranchResponse represents the API response for a branch
type BranchResponse struct {
	ID       string `json:"id"`
	CommitID string `json:"commit_id"`
}

func (r *BranchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (r *BranchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_branch.BranchResourceSchema(ctx)
}

func (r *BranchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_branch.BranchModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	createReq := BranchCreateRequest{
		Name:   data.Name.ValueString(),
		Source: data.Source.ValueString(),
	}

	tflog.Debug(ctx, "Creating branch", map[string]any{
		"repository": repository,
		"name":       createReq.Name,
		"source":     createReq.Source,
	})

	// LakeFS branch creation returns a plain string (the commit ID), not JSON
	commitID, err := client.PostRaw(ctx, fmt.Sprintf("/repositories/%s/branches", repository), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create branch: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(fmt.Sprintf("%s/%s", repository, createReq.Name))
	data.CommitId = types.StringValue(commitID)
	data.Branch = types.StringValue(createReq.Name)

	tflog.Trace(ctx, "Created branch", map[string]any{
		"id":        data.Id.ValueString(),
		"commit_id": commitID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_branch.BranchModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	branchName := data.Name.ValueString()
	if branchName == "" {
		branchName = data.Branch.ValueString()
	}

	var result BranchResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/branches/%s", repository, branchName), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read branch: %s", err))
		return
	}

	// Map response to state
	data.CommitId = types.StringValue(result.CommitID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_branch.BranchModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Branches in LakeFS cannot be updated - name changes require delete/recreate
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_branch.BranchModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)

	repository := data.Repository.ValueString()
	branchName := data.Name.ValueString()
	if branchName == "" {
		branchName = data.Branch.ValueString()
	}

	tflog.Debug(ctx, "Deleting branch", map[string]any{
		"repository": repository,
		"branch":     branchName,
	})

	err := client.Delete(ctx, fmt.Sprintf("/repositories/%s/branches/%s", repository, branchName))
	if err != nil {
		if !IsNotFound(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete branch: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "Deleted branch", map[string]any{
		"repository": repository,
		"branch":     branchName,
	})
}

func (r *BranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: repository/branch
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'repository/branch', got: %s", req.ID),
		)
		return
	}

	client := NewAPIClient(r.client)
	repository := parts[0]
	branchName := parts[1]

	var result BranchResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/branches/%s", repository, branchName), &result)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to import branch %s: %s", req.ID, err))
		return
	}

	var data resource_branch.BranchModel
	data.Id = types.StringValue(req.ID)
	data.Repository = types.StringValue(repository)
	data.Name = types.StringValue(branchName)
	data.Branch = types.StringValue(branchName)
	data.CommitId = types.StringValue(result.CommitID)
	data.Source = types.StringValue("") // Source is not retrievable after creation

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
