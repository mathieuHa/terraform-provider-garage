data "garage_admin_token" "current" {}

output "provider_scope" {
  value = data.garage_admin_token.current.scope
}
