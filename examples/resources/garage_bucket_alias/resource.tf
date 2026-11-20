resource "garage_bucket_alias" "legacy_name" {
  bucket_id = garage_bucket.app.id
  alias     = "old-app-data"
}

resource "garage_bucket_alias" "per_key" {
  bucket_id     = garage_bucket.app.id
  access_key_id = garage_key.app.id
  alias         = "data"
}
