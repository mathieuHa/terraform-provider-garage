data "garage_cluster_status" "this" {}

output "node_ids" {
  value = { for n in data.garage_cluster_status.this.nodes : n.hostname => n.id }
}
