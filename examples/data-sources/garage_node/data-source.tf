data "garage_node" "self" {}

output "garage_version" {
  value = data.garage_node.self.garage_version
}
