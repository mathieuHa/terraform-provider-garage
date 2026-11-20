data "garage_bucket" "app" {
  global_alias = "app-data"
}

output "app_usage_bytes" {
  value = data.garage_bucket.app.bytes
}
