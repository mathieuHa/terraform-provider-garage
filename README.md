# Terraform provider for Garage

An OpenTofu and Terraform provider for [Garage](https://garagehq.deuxfleurs.fr/), the self-hosted S3-compatible object store.

**Documentation: <https://mathieuha.github.io/terraform-provider-garage/>**: [quickstart](https://mathieuha.github.io/terraform-provider-garage/quickstart/), [guides](https://mathieuha.github.io/terraform-provider-garage/guides/concepts/), [supported resources](https://mathieuha.github.io/terraform-provider-garage/resources/) and [API coverage](https://mathieuha.github.io/terraform-provider-garage/coverage/).

It manages everything Garage's admin API v2 can configure:

| Resource | Manages |
| --- | --- |
| `garage_bucket` | Bucket, global alias, quotas, website hosting with routing rules, CORS rules, lifecycle rules |
| `garage_bucket_alias` | Additional global aliases and per-key local aliases |
| `garage_key` | Access keys: generated or imported with known credentials, expiration, bucket creation right |
| `garage_bucket_permission` | `read` / `write` / `owner` of a key on a bucket |
| `garage_admin_token` | Admin API tokens restricted to a list of endpoints, with expiration |
| `garage_cluster_layout` | Node roles (zone, capacity, tags) and zone redundancy, staged and applied in one step |

Data sources: `garage_bucket`, `garage_buckets`, `garage_key`, `garage_keys`, `garage_admin_token`, `garage_cluster_health`, `garage_cluster_status`, `garage_cluster_layout`, `garage_node`.

Requirements: Garage 2.3.0 or later, OpenTofu or Terraform 1.x. The acceptance tests run against Garage 2.3.0 and 2.5.0.

## Usage

```hcl
terraform {
  required_providers {
    garage = {
      source = "mathieuha/garage"
    }
  }
}

provider "garage" {
  endpoint = "http://127.0.0.1:3903"
  token    = var.garage_admin_token
}

resource "garage_bucket" "app" {
  global_alias = "app-data"
  max_size     = 107374182400

  lifecycle_rules = [
    {
      filter                                 = { prefix = "tmp/" }
      expiration_days                        = 7
      abort_incomplete_multipart_upload_days = 1
    },
  ]
}

resource "garage_key" "app" {
  name = "app"
}

resource "garage_bucket_permission" "app" {
  bucket_id     = garage_bucket.app.id
  access_key_id = garage_key.app.id
  read          = true
  write         = true
}
```

The endpoint and token can also come from `GARAGE_ENDPOINT` and `GARAGE_TOKEN`. For an admin API behind TLS with a private CA, set `ca_cert_file`.

See [`docs/`](docs/) for every attribute, and [`examples/`](examples/) for more configurations.

## Behaviour worth knowing

- **Website hosting and quotas are authoritative.** Without a `website` block, website hosting is disabled. Without `max_size`, there is no size quota.
- **CORS and lifecycle rules are authoritative only when set.** If `cors_rules` or `lifecycle_rules` is absent, rules set through the S3 API are left alone and never show as drift. Set it to `[]` to remove every rule.
- **Lifecycle filters are flat.** Write `filter = { prefix = "tmp/", object_size_greater_than = 1024 }`, and the provider wraps several conditions in the `And` element that Garage requires.
- **Permissions are per key and bucket.** Garage has no IAM policies, so `garage_bucket_permission` is the only access control. Destroying it revokes the key's access to the bucket.
- **Secrets are in the state.** The secret of `garage_key` is read back from Garage, so an imported key is complete. The secret of `garage_admin_token` is only returned at creation.
- **Imported key IDs are final.** Garage never accepts an access key ID again once its key is deleted, even through `ImportKey`.
- **The cluster layout is authoritative, and destroying it changes nothing.** Nodes missing from `roles` are removed from the layout, and changes staged by someone else are reverted first. Destroying the resource only removes it from the state, because removing every role would make the cluster unusable.
- **Buckets must be empty to be deleted.** Garage refuses to delete a bucket that still contains objects, and the admin API cannot delete objects.

Imperative admin operations are not exposed: connecting nodes, skipping dead nodes, repairs, snapshots, workers, block operations, purging incomplete uploads.

## Development

```sh
make build       # build the provider
make test        # unit tests
make testacc     # start a throwaway Garage in Docker and run the acceptance tests with OpenTofu
make testacc-down
make docs        # regenerate docs/ from the schema, templates/ and examples/
make site        # build the documentation site into site/build/public (Docker)
make site-serve  # preview the site on http://127.0.0.1:8000
make lint
```

The documentation site is built by `site/gen.py` with [Zensical](https://zensical.org/), from `docs/`, `examples/` and the pages in `site/content/`. It is published to GitHub Pages on every push to `main`. The guide examples in `examples/guides/` are complete configurations: apply them against the test Garage before changing them.

`make testacc` uses `scripts/testacc-garage.sh`, which starts `dxflrs/garage:v2.5.0` on ports 13900 and 13903. Set `GARAGE_IMAGE` to test another version. To use a local build, run `make install` and follow the `dev_overrides` instructions it prints.

## Releasing

Pushing a `v*` tag runs GoReleaser, which builds, signs and publishes the release. The repository needs the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets. Releases are signed with the key in [`signing-key.asc`](signing-key.asc) (RSA 4096, fingerprint `9C0F 4550 B465 8640 4262  B87E 051F D91A 3514 DDDA`). The Terraform Registry picks up releases once the provider is registered there. The OpenTofu Registry needs a one-time submission to [opentofu/registry](https://github.com/opentofu/registry), with the public signing key.

## License

Apache 2.0, see [LICENSE](LICENSE).
