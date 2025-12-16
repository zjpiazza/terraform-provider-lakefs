# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - YYYY-MM-DD

### Added

- Initial release of the LakeFS Terraform Provider
- **Resources:**
  - `lakefs_repository` - Manage LakeFS repositories
  - `lakefs_branch` - Manage branches within repositories
  - `lakefs_tag` - Manage tags within repositories
  - `lakefs_branch_protection` - Manage branch protection rules
  - `lakefs_user` - Manage users (Enterprise/Cloud only)
  - `lakefs_group` - Manage groups (Enterprise/Cloud only)
  - `lakefs_policy` - Manage policies (Enterprise/Cloud only)
  - `lakefs_group_membership` - Manage group memberships (Enterprise/Cloud only)
  - `lakefs_group_policy_attachment` - Attach policies to groups (Enterprise/Cloud only)
  - `lakefs_user_policy_attachment` - Attach policies to users (Enterprise/Cloud only)
- **Data Sources:**
  - `lakefs_repository` - Query repository information
  - `lakefs_branch` - Query branch information
  - `lakefs_commit` - Query commit information
  - `lakefs_current_user` - Query current authenticated user
  - `lakefs_user` - Query user information (Enterprise/Cloud only)
  - `lakefs_group` - Query group information (Enterprise/Cloud only)

### Notes

- RBAC resources (User, Group, Policy, and related resources) are only available in LakeFS Enterprise or LakeFS Cloud
- OSS resources (Repository, Branch, Tag, Branch Protection) are available in all LakeFS editions

[Unreleased]: https://github.com/zjpiazza/terraform-provider-lakefs/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/zjpiazza/terraform-provider-lakefs/releases/tag/v0.1.0
