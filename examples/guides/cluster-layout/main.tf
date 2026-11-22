data "garage_cluster_status" "this" {}

locals {
  zones = {
    "garage-a1" = "site-a"
    "garage-a2" = "site-a"
    "garage-b1" = "site-b"
  }
}

resource "garage_cluster_layout" "this" {
  roles = {
    for node in data.garage_cluster_status.this.nodes : node.id => {
      zone     = lookup(local.zones, node.hostname, "site-a")
      capacity = 2000000000000
      tags     = [node.hostname]
    }
  }
}

data "garage_cluster_health" "this" {
  depends_on = [garage_cluster_layout.this]
}

output "health" {
  value = data.garage_cluster_health.this.status
}
