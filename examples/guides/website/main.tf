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
      {
        condition = { http_error_code_returned_equals = 404, key_prefix_equals = "docs/" }
        redirect  = { replace_key_with = "docs/index.html" }
      },
    ]
  }
}

resource "garage_key" "deploy" {
  name = "www-deploy"
}

resource "garage_bucket_permission" "deploy" {
  bucket_id     = garage_bucket.site.id
  access_key_id = garage_key.deploy.id
  read          = true
  write         = true
}
