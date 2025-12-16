resource "lakefs_branch" "develop" {
  repository = lakefs_repository.example.id
  name       = "develop"
  source     = "main"
}
