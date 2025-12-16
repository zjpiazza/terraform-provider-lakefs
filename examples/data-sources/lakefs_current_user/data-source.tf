data "lakefs_current_user" "me" {}

output "my_user_id" {
  value = data.lakefs_current_user.me.id
}

output "my_email" {
  value = data.lakefs_current_user.me.email
}
