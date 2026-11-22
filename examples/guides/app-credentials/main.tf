locals {
  apps = {
    "billing" = { buckets = { "billing-invoices" = "rw", "billing-exports" = "rw" } }
    "reports" = { buckets = { "billing-exports" = "ro" } }
  }
  buckets = toset(flatten([for app in values(local.apps) : keys(app.buckets)]))
  grants = merge([
    for app, cfg in local.apps : {
      for bucket, mode in cfg.buckets : "${app}/${bucket}" => { app = app, bucket = bucket, write = mode == "rw" }
    }
  ]...)
}

resource "garage_bucket" "this" {
  for_each     = local.buckets
  global_alias = each.key
}

resource "garage_key" "app" {
  for_each = local.apps
  name     = each.key
}

resource "garage_bucket_permission" "this" {
  for_each      = local.grants
  bucket_id     = garage_bucket.this[each.value.bucket].id
  access_key_id = garage_key.app[each.value.app].id
  read          = true
  write         = each.value.write
}

resource "vault_kv_secret_v2" "s3" {
  for_each = garage_key.app
  mount    = "secret"
  name     = "garage/${each.key}"
  data_json = jsonencode({
    access_key = each.value.id
    secret_key = each.value.secret_access_key
    endpoint   = "https://s3.example.com"
    region     = "garage"
  })
}
