# Application credentials in OpenBao or Vault

Give each application its own key, with read-write or read-only access to the buckets it needs, and publish the credentials where the application reads them.

```terraform
--8<-- "examples/guides/app-credentials/main.tf"
```

- `local.apps` is the only thing to edit: each application lists its buckets with `rw` or `ro`.
- The buckets are the union of all the buckets mentioned, so a bucket shared by two applications is created once.
- `garage_bucket_permission` grants `read` to every pair, and `write` only for `rw`.
- `vault_kv_secret_v2` publishes one secret per application. The `hashicorp/vault` provider works the same with OpenBao.

Applications then read `access_key` and `secret_key` from `secret/garage/<app>`, for example with the External Secrets Operator or Vault Agent.

!!! note "The secret is in the state"

    `garage_key.secret_access_key` is stored in the state, as with any key created by OpenTofu. Keep the state in an encrypted backend. Garage returns the secret on every read, so an imported key is complete too.

To rotate a key, replace it: `tofu apply -replace='garage_key.app["billing"]'`. The permissions and the secret follow, because they reference the key.
