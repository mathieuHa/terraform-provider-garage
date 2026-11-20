resource "garage_bucket" "app" {
  global_alias = "app-data"
  max_size     = 107374182400
  max_objects  = 1000000

  cors_rules = [
    {
      allowed_origins = ["https://app.example.com"]
      allowed_methods = ["GET", "PUT"]
      allowed_headers = ["*"]
      expose_headers  = ["ETag"]
      max_age_seconds = 3600
    },
  ]

  lifecycle_rules = [
    {
      id              = "expire-tmp"
      filter          = { prefix = "tmp/" }
      expiration_days = 7
    },
    {
      id                                     = "abort-uploads"
      abort_incomplete_multipart_upload_days = 1
    },
  ]
}

resource "garage_bucket" "site" {
  global_alias = "www.example.com"

  website = {
    index_document = "index.html"
    error_document = "404.html"
    routing_rules = [
      {
        condition = { key_prefix_equals = "blog/" }
        redirect  = { replace_key_prefix_with = "posts/", http_redirect_code = 301 }
      },
    ]
  }
}
