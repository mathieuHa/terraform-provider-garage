resource "garage_admin_token" "monitoring" {
  name  = "monitoring"
  scope = ["GetClusterHealth", "GetClusterStatus", "GetClusterStatistics"]
}

resource "garage_admin_token" "tofu_buckets" {
  name       = "tofu-buckets"
  expiration = "2027-01-01T00:00:00Z"
  scope = [
    "ListBuckets", "GetBucketInfo", "CreateBucket", "UpdateBucket", "DeleteBucket",
    "AddBucketAlias", "RemoveBucketAlias", "AllowBucketKey", "DenyBucketKey",
    "ListKeys", "GetKeyInfo", "CreateKey", "ImportKey", "UpdateKey", "DeleteKey",
  ]
}
