resource "garage_admin_token" "tofu" {
  name       = "tofu-buckets-and-keys"
  expiration = "2027-12-31T00:00:00Z"
  scope = [
    "GetClusterHealth",
    "ListBuckets", "GetBucketInfo", "CreateBucket", "UpdateBucket", "DeleteBucket",
    "AddBucketAlias", "RemoveBucketAlias",
    "AllowBucketKey", "DenyBucketKey",
    "ListKeys", "GetKeyInfo", "CreateKey", "ImportKey", "UpdateKey", "DeleteKey",
  ]
}

resource "garage_admin_token" "monitoring" {
  name  = "monitoring"
  scope = ["GetClusterHealth", "GetClusterStatus", "GetClusterStatistics"]
}

output "tofu_token" {
  value     = garage_admin_token.tofu.secret_token
  sensitive = true
}
