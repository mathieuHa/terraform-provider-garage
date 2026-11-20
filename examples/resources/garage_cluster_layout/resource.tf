data "garage_cluster_status" "this" {}

resource "garage_cluster_layout" "this" {
  roles = {
    for node in data.garage_cluster_status.this.nodes : node.id => {
      zone     = startswith(node.hostname, "paris-") ? "paris" : "lyon"
      capacity = 1000000000000
      tags     = [node.hostname]
    }
  }
  zone_redundancy = 2
}
