import {
  to = garage_bucket.legacy
  id = "legacy-data"
}

import {
  to = garage_key.legacy_app
  id = var.legacy_key_id
}

import {
  to = garage_bucket_permission.legacy_app
  id = "${garage_bucket.legacy.id}/${var.legacy_key_id}"
}

variable "legacy_key_id" {
  type = string
}

resource "garage_bucket" "legacy" {
  global_alias = "legacy-data"
}

resource "garage_key" "legacy_app" {
  name = "legacy-app"
}

resource "garage_bucket_permission" "legacy_app" {
  bucket_id     = garage_bucket.legacy.id
  access_key_id = garage_key.legacy_app.id
  read          = true
  write         = true
}
