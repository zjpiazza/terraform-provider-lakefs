// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CurrentUserDataSource{}

func NewCurrentUserDataSource() datasource.DataSource {
	return &CurrentUserDataSource{}
}

// CurrentUserDataSource defines the data source implementation.
type CurrentUserDataSource struct {
	client *LakeFSClient
}

// CurrentUserModel describes the data source data model.
type CurrentUserModel struct {
	Id           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FriendlyName types.String `tfsdk:"friendly_name"`
	CreationDate types.Int64  `tfsdk:"creation_date"`
}

// CurrentUserResponse represents the API response for current user
type CurrentUserResponse struct {
	User struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		FriendlyName string `json:"friendly_name"`
		CreationDate int64  `json:"creation_date"`
	} `json:"user"`
}

func (d *CurrentUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user"
}

func (d *CurrentUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about the currently authenticated user.",
		MarkdownDescription: `Retrieves information about the currently authenticated user.

## Example Usage

` + "```hcl" + `
data "lakefs_current_user" "me" {}

output "current_user_id" {
  value = data.lakefs_current_user.me.id
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the current user.",
			},
			"email": schema.StringAttribute{
				Computed:    true,
				Description: "The email address of the current user.",
			},
			"friendly_name": schema.StringAttribute{
				Computed:    true,
				Description: "A shorter, more friendly name for the user.",
			},
			"creation_date": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix epoch timestamp when the user was created.",
			},
		},
	}
}

func (d *CurrentUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CurrentUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CurrentUserModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := NewAPIClient(d.client)

	var result CurrentUserResponse
	err := client.Get(ctx, "/user", &result)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read current user: %s", err))
		return
	}

	// Map response to state
	data.Id = types.StringValue(result.User.ID)
	data.Email = types.StringValue(result.User.Email)
	data.FriendlyName = types.StringValue(result.User.FriendlyName)
	data.CreationDate = types.Int64Value(result.User.CreationDate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
