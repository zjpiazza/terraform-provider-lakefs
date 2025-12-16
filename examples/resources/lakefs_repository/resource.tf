resource "lakefs_repository" "example" {
  name              = "my-repository"
  storage_namespace = "s3://my-bucket/lakefs/my-repository"
  default_branch    = "main"
}
