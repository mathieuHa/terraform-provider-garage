resource "garage_bucket" "uploads" {
  global_alias = "uploads"

  cors_rules = [
    {
      id              = "web-app"
      allowed_origins = ["https://app.example.com"]
      allowed_methods = ["GET", "PUT", "POST", "DELETE", "HEAD"]
      allowed_headers = ["*"]
      expose_headers  = ["ETag"]
      max_age_seconds = 3600
    },
  ]

  lifecycle_rules = [
    {
      id              = "expire-tmp"
      filter          = { prefix = "tmp/" }
      expiration_days = 1
    },
    {
      id              = "expire-large-exports"
      filter          = { prefix = "exports/", object_size_greater_than = 104857600 }
      expiration_days = 30
    },
    {
      id                                     = "abort-stalled-uploads"
      abort_incomplete_multipart_upload_days = 2
    },
  ]
}

resource "garage_bucket" "app_managed" {
  global_alias = "app-managed"
}
