terraform {
  required_providers {
    garage = {
      source = "mathieuha/garage"
    }
  }
}

provider "garage" {
  endpoint = "https://garage-admin.example.com"
  token    = var.garage_admin_token
}
