data "garage_cluster_health" "this" {}

check "garage_healthy" {
  assert {
    condition     = data.garage_cluster_health.this.status == "healthy"
    error_message = "Garage is ${data.garage_cluster_health.this.status}."
  }
}
