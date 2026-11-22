# Importing existing objects

Bring buckets, keys and permissions created with the `garage` CLI, or by another tool, under management without recreating them.

```terraform
--8<-- "examples/guides/import/main.tf"
```

```bash
tofu plan -var legacy_key_id=GK1234567890abcdef12345678
```

The plan shows three imports and no other change when the configuration matches what exists. Apply it, then remove the `import` blocks.

| Resource | Import ID |
|---|---|
| `garage_bucket` | Bucket ID or global alias |
| `garage_bucket_alias` | `<bucket_id>/<alias>`, or `<bucket_id>/<access_key_id>/<alias>` for a local alias |
| `garage_key` | Access key ID |
| `garage_bucket_permission` | `<bucket_id>/<access_key_id>` |
| `garage_admin_token` | Token ID |
| `garage_cluster_layout` | `cluster_layout` |

What an import brings in:

- **Keys come with their secret.** Garage returns it on read, so the imported key can be published to a secret store like a new one, and the credentials do not change.
- **CORS and lifecycle rules are imported** when the bucket has some, and become managed.
- **Admin tokens come without their secret**, which Garage only returns at creation.

## Moving from another provider

To switch from another Garage provider, remove its resources from the state with `tofu state rm`, then import them under the same addresses with this provider. Nothing changes in Garage, and the key secrets stay the same.

## Keys from another Garage cluster

`garage_key` can also create a key with known credentials, by setting `id` and `secret_access_key`. Use it to recreate keys from another Garage cluster with the same credentials. Both values must be in the Garage format, and Garage never accepts an ID again once its key has been deleted.
