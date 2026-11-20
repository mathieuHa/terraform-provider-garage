resource "garage_bucket_permission" "app" {
  bucket_id     = garage_bucket.app.id
  access_key_id = garage_key.app.id
  read          = true
  write         = true
}

resource "garage_bucket_permission" "backup_reader" {
  bucket_id     = garage_bucket.app.id
  access_key_id = garage_key.backup.id
  read          = true
}
