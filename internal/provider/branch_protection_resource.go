// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BranchProtectionResource{}
var _ resource.ResourceWithImportState = &BranchProtectionResource{}

func NewBranchProtectionResource() resource.Resource {
	return &BranchProtectionResource{}
}

// BranchProtectionResource defines the resource implementation.
type BranchProtectionResource struct {
	client *LakeFSClient
}

// BranchProtectionModel describes the resource data model.
type BranchProtectionModel struct {
	Repository types.String `tfsdk:"repository"`
	Id         types.String `tfsdk:"id"`
	Rules      types.List   `tfsdk:"rules"`
}

// BranchProtectionRule represents a branch protection rule
type BranchProtectionRule struct {
	Pattern string `json:"pattern"`
}

// BranchProtectionRulesResponse represents the API response
type BranchProtectionRulesResponse []BranchProtectionRule

func (r *BranchProtectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_protection"
}

func (r *BranchProtectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages branch protection rules for a LakeFS repository.",
		MarkdownDescription: `Manages branch protection rules for a LakeFS repository.

Branch protection rules prevent direct commits to matching branches, requiring changes to be merged via merge operations.

## Example Usage

` + "```hcl" + `
resource "lakefs_branch_protection" "main" {
  repository = lakefs_repository.example.id

  rules = [
    { pattern = "main" },
    { pattern = "release-*" }
  ]
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"repository": schema.StringAttribute{
				Required:    true,
				Description: "The repository ID to apply branch protection rules to.",
			},
			"rules": schema.ListNestedAttribute{
				Required:    true,
				Description: "List of branch protection rules. Each rule contains a pattern to match branch names.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pattern": schema.StringAttribute{
							Required:    true,
							Description: "Pattern to match branch names (supports wildcards, e.g., 'release-*').",
						},
					},
				},
			},
		},
	}
}

func (r *BranchProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BranchProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BranchProtectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)
	repository := data.Repository.ValueString()

	// Extract rules from the plan
	rules, diags := extractBranchProtectionRules(ctx, data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating branch protection rules", map[string]any{
		"repository": repository,
		"rules":      rules,
	})

	// LakeFS uses PUT to set branch protection rules
	err := client.Put(ctx, fmt.Sprintf("/repositories/%s/settings/branch_protection", repository), rules, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create branch protection rules: %s", err))
		return
	}

	// Set computed fields
	data.Id = types.StringValue(repository)

	tflog.Trace(ctx, "Created branch protection rules", map[string]any{"repository": repository})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BranchProtectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)
	repository := data.Repository.ValueString()

	var result BranchProtectionRulesResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/settings/branch_protection", repository), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read branch protection rules: %s", err))
		return
	}

	// Convert rules to Terraform types
	rulesList, diags := branchProtectionRulesToTerraformList(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Rules = rulesList
	data.Id = types.StringValue(repository)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BranchProtectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)
	repository := data.Repository.ValueString()

	// Extract rules from the plan
	rules, diags := extractBranchProtectionRules(ctx, data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating branch protection rules", map[string]any{
		"repository": repository,
		"rules":      rules,
	})

	err := client.Put(ctx, fmt.Sprintf("/repositories/%s/settings/branch_protection", repository), rules, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update branch protection rules: %s", err))
		return
	}

	data.Id = types.StringValue(repository)

	tflog.Trace(ctx, "Updated branch protection rules", map[string]any{"repository": repository})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BranchProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BranchProtectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(r.client)
	repository := data.Repository.ValueString()

	tflog.Debug(ctx, "Deleting branch protection rules", map[string]any{"repository": repository})

	// Delete by setting empty rules
	err := client.Put(ctx, fmt.Sprintf("/repositories/%s/settings/branch_protection", repository), []BranchProtectionRule{}, nil)
	if err != nil {
		if !IsNotFound(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete branch protection rules: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "Deleted branch protection rules", map[string]any{"repository": repository})
}

func (r *BranchProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client := NewAPIClient(r.client)
	repository := req.ID

	var result BranchProtectionRulesResponse
	err := client.Get(ctx, fmt.Sprintf("/repositories/%s/settings/branch_protection", repository), &result)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to import branch protection rules for %s: %s", repository, err))
		return
	}

	rulesList, diags := branchProtectionRulesToTerraformList(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data BranchProtectionModel
	data.Id = types.StringValue(repository)
	data.Repository = types.StringValue(repository)
	data.Rules = rulesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// extractBranchProtectionRules extracts rules from Terraform types
func extractBranchProtectionRules(ctx context.Context, rulesList types.List) ([]BranchProtectionRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	if rulesList.IsNull() || rulesList.IsUnknown() {
		return nil, diags
	}

	var rules []BranchProtectionRule
	elements := rulesList.Elements()

	for _, elem := range elements {
		obj := elem.(types.Object)
		attrs := obj.Attributes()

		rule := BranchProtectionRule{
			Pattern: attrs["pattern"].(types.String).ValueString(),
		}
		rules = append(rules, rule)
	}

	return rules, diags
}

// branchProtectionRulesToTerraformList converts rules to Terraform types.List
func branchProtectionRulesToTerraformList(ctx context.Context, rules []BranchProtectionRule) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	ruleAttrTypes := map[string]attr.Type{
		"pattern": types.StringType,
	}

	if len(rules) == 0 {
		return types.ListValueMust(
			types.ObjectType{AttrTypes: ruleAttrTypes},
			[]attr.Value{},
		), diags
	}

	var ruleValues []attr.Value
	for _, rule := range rules {
		ruleObj, _ := types.ObjectValue(
			ruleAttrTypes,
			map[string]attr.Value{
				"pattern": types.StringValue(rule.Pattern),
			},
		)
		ruleValues = append(ruleValues, ruleObj)
	}

	rulesList, _ := types.ListValue(
		types.ObjectType{AttrTypes: ruleAttrTypes},
		ruleValues,
	)

	return rulesList, diags
}
