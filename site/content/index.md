# Garage provider for OpenTofu and Terraform

Manage [Garage](https://garagehq.deuxfleurs.fr/) object storage as code: buckets with quotas, website hosting, CORS and lifecycle rules, access keys and their permissions, scoped admin tokens and the cluster layout, through the Garage admin API v2.

<div class="grid cards" markdown>

-   **{{ resources }} resources, {{ data_sources }} data sources**

    ---

    Every setting the admin API can configure, with drift detection and import for every resource.

    [→ Supported resources](resources.md)

-   **Five minutes to the first bucket**

    ---

    Point the provider at the admin API and create a bucket, a key and its permission.

    [→ Quickstart](quickstart.md)

-   **Tested recipes**

    ---

    Application credentials in OpenBao or Vault, static websites, lifecycle and CORS rules, least-privilege tokens, multi-site layouts.

    [→ Guides](guides/concepts.md)

-   **Signed releases**

    ---

    Published as `mathieuha/garage`, every release signed and tested against Garage 2.3 and 2.5 with OpenTofu and Terraform.

    [→ Releases](https://github.com/mathieuHa/terraform-provider-garage/releases)

</div>

## Install

```terraform
terraform {
  required_providers {
    garage = {
      source  = "mathieuha/garage"
      version = "~> 0.1"
    }
  }
}

provider "garage" {
  endpoint = "http://garage.example.com:3903"
  token    = var.garage_admin_token
}
```

## What you can manage

| Area | Resources |
|---|---|
| Buckets | Buckets with global aliases, size and object quotas, website hosting with redirect rules, CORS rules, lifecycle rules; extra global and per-key local aliases |
| Access | Access keys (generated or imported), expiration, the right to create buckets; read, write and owner permissions per key and bucket |
| Administration | Admin API tokens restricted to a list of endpoints, with expiration |
| Cluster | Node roles (zone, capacity, tags) and zone redundancy, applied as one layout version; health, status and node information |

Garage has no IAM, so there are no users or policies to manage: see [Concepts](guides/concepts.md) for how access works.
