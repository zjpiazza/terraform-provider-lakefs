resource "lakefs_tag" "release" {
  repository = lakefs_repository.example.id
  id         = "v1.0.0"
  ref        = "main"
}
