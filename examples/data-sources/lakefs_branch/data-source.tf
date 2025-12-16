data "lakefs_branch" "main" {
  repository = "my-repository"
  branch     = "main"
}

output "current_commit" {
  value = data.lakefs_branch.main.commit_id
}
