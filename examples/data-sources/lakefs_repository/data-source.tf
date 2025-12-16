data "lakefs_repository" "example" {
  repository = "my-repository"
}

output "storage_namespace" {
  value = data.lakefs_repository.example.storage_namespace
}
