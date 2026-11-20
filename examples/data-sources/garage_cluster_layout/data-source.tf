data "garage_cluster_layout" "this" {}

output "usable_capacity" {
  value = sum(data.garage_cluster_layout.this.roles[*].usable_capacity)
}
