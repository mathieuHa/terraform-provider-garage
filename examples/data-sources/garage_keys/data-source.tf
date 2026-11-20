data "garage_keys" "all" {}

output "expired_keys" {
  value = [for k in data.garage_keys.all.keys : k.name if k.expired]
}
