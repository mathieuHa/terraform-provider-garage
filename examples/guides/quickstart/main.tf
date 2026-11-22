terraform {
  required_providers {
    garage = {
      source  = "mathieuha/garage"
      version = "~> 0.1"
    }
  }
}

provider "garage" {}

resource "garage_bucket" "photos" {
  global_alias = "photos"
  max_size     = 10737418240
}

resource "garage_key" "photos_app" {
  name = "photos-app"
}

resource "garage_bucket_permission" "photos_app" {
  bucket_id     = garage_bucket.photos.id
  access_key_id = garage_key.photos_app.id
  read          = true
  write         = true
}

output "access_key_id" {
  value = garage_key.photos_app.id
}

output "secret_access_key" {
  value     = garage_key.photos_app.secret_access_key
  sensitive = true
}
