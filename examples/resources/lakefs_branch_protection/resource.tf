resource "lakefs_branch_protection" "main" {
  repository = lakefs_repository.example.id

  rules = [
    { pattern = "main" },
    { pattern = "release-*" }
  ]
}
