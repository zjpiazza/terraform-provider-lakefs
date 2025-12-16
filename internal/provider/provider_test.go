// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"lakefs": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Check for required environment variables
	if v := os.Getenv("LAKEFS_ENDPOINT"); v == "" {
		t.Fatal("LAKEFS_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("LAKEFS_ACCESS_KEY_ID"); v == "" {
		t.Fatal("LAKEFS_ACCESS_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("LAKEFS_SECRET_ACCESS_KEY"); v == "" {
		t.Fatal("LAKEFS_SECRET_ACCESS_KEY must be set for acceptance tests")
	}
}

func TestAccRepositoryResource(t *testing.T) {
	// Use unique name with timestamp to avoid conflicts with previous test runs
	repoName := fmt.Sprintf("testrepo%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRepositoryResourceConfig(repoName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lakefs_repository.test", "name", repoName),
					resource.TestCheckResourceAttr("lakefs_repository.test", "default_branch", "main"),
					resource.TestCheckResourceAttrSet("lakefs_repository.test", "creation_date"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "lakefs_repository.test",
				ImportState:       true,
				ImportStateVerify: true,
				// sample_data is not returned by the API
				ImportStateVerifyIgnore: []string{"sample_data"},
			},
		},
	})
}

func testAccRepositoryResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}
`, name)
}

func TestAccBranchResource(t *testing.T) {
	// Use unique names with timestamp to avoid conflicts
	repoName := fmt.Sprintf("branchtestrepo%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBranchResourceConfig(repoName, "testbranch"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lakefs_branch.test", "name", "testbranch"),
					resource.TestCheckResourceAttrSet("lakefs_branch.test", "commit_id"),
				),
			},
		},
	})
}

func testAccBranchResourceConfig(repoName, branchName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

resource "lakefs_branch" "test" {
  repository = lakefs_repository.test.id
  name       = %[2]q
  source     = "main"
}
`, repoName, branchName)
}

func TestAccCurrentUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCurrentUserDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.lakefs_current_user.test", "id"),
				),
			},
		},
	})
}

const testAccCurrentUserDataSourceConfig = `
data "lakefs_current_user" "test" {}
`

// =====================
// Tag Resource Tests
// =====================

func TestAccTagResource(t *testing.T) {
	repoName := fmt.Sprintf("tagtestrepo%d", time.Now().UnixNano())
	tagName := "v1.0.0"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagResourceConfig(repoName, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lakefs_tag.test", "id", tagName),
					resource.TestCheckResourceAttrSet("lakefs_tag.test", "commit_id"),
				),
			},
		},
	})
}

func testAccTagResourceConfig(repoName, tagName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

resource "lakefs_tag" "test" {
  repository = lakefs_repository.test.id
  id         = %[2]q
  ref        = "main"
}
`, repoName, tagName)
}

// =====================
// Branch Protection Resource Tests
// =====================

func TestAccBranchProtectionResource(t *testing.T) {
	repoName := fmt.Sprintf("bptestrepo%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBranchProtectionResourceConfig(repoName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lakefs_branch_protection.test", "repository", repoName),
				),
			},
		},
	})
}

func testAccBranchProtectionResourceConfig(repoName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

resource "lakefs_branch_protection" "test" {
  repository = lakefs_repository.test.id
  rules = [
    { pattern = "main" }
  ]
}
`, repoName)
}

// =====================
// Repository Data Source Tests
// =====================

func TestAccRepositoryDataSource(t *testing.T) {
	repoName := fmt.Sprintf("dsrepo%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryDataSourceConfig(repoName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.lakefs_repository.test", "default_branch", "main"),
					resource.TestCheckResourceAttrSet("data.lakefs_repository.test", "storage_namespace"),
				),
			},
		},
	})
}

func testAccRepositoryDataSourceConfig(repoName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

data "lakefs_repository" "test" {
  repository = lakefs_repository.test.id
}
`, repoName)
}

// =====================
// Branch Data Source Tests
// =====================

func TestAccBranchDataSource(t *testing.T) {
	repoName := fmt.Sprintf("dsbranch%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBranchDataSourceConfig(repoName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.lakefs_branch.test", "branch", "main"),
					resource.TestCheckResourceAttrSet("data.lakefs_branch.test", "commit_id"),
				),
			},
		},
	})
}

func testAccBranchDataSourceConfig(repoName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

data "lakefs_branch" "test" {
  repository = lakefs_repository.test.id
  branch     = "main"
}
`, repoName)
}

// =====================
// Commit Data Source Tests
// =====================

func TestAccCommitDataSource(t *testing.T) {
	repoName := fmt.Sprintf("dscommit%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCommitDataSourceConfig(repoName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.lakefs_commit.test", "id"),
					resource.TestCheckResourceAttrSet("data.lakefs_commit.test", "creation_date"),
				),
			},
		},
	})
}

func testAccCommitDataSourceConfig(repoName string) string {
	return fmt.Sprintf(`
resource "lakefs_repository" "test" {
  name              = %[1]q
  storage_namespace = "s3://lakefs-data/%[1]s"
  default_branch    = "main"
}

data "lakefs_branch" "main" {
  repository = lakefs_repository.test.id
  branch     = "main"
}

data "lakefs_commit" "test" {
  repository = lakefs_repository.test.id
  commit_id  = data.lakefs_branch.main.commit_id
}
`, repoName)
}
