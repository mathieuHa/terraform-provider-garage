data "garage_buckets" "all" {}

output "bucket_names" {
  value = flatten(data.garage_buckets.all.buckets[*].global_aliases)
}
