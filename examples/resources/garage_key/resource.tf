resource "garage_key" "app" {
  name = "app"
}

resource "garage_key" "ci" {
  name                = "ci"
  allow_create_bucket = true
  expiration          = "2027-01-01T00:00:00Z"
}

resource "garage_key" "migrated" {
  name              = "migrated-from-minio"
  id                = "GK31c2f218a2e44f485b94239e"
  secret_access_key = var.migrated_secret
}

resource "vault_kv_secret_v2" "app" {
  mount = "secret"
  name  = "s3/app"
  data_json = jsonencode({
    access_key = garage_key.app.id
    secret_key = garage_key.app.secret_access_key
  })
}
