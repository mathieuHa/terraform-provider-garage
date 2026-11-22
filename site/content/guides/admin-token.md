# Least-privilege admin token

The `admin_token` of the daemon configuration can call every admin endpoint, including the ones that change the cluster layout. Give OpenTofu a token that can only do what your configuration needs.

```terraform
--8<-- "examples/guides/admin-token/main.tf"
```

- `tofu` can manage buckets, aliases, keys and permissions, and nothing else. It expires at the end of 2027.
- `monitoring` can only read the cluster health and status, for a monitoring probe.

`scope` takes endpoint names; the [API coverage](../coverage.md) page lists which ones each resource calls. `["*"]` allows every endpoint.

## Switching OpenTofu to the new token

1. Apply this configuration once with the daemon's `admin_token`.
2. Store the secret: `tofu output -raw tofu_token`, then put it in your secret store.
3. Use it as `GARAGE_TOKEN` for your other configurations.

Keep the configuration that creates the token separate, applied with the daemon's token: a token cannot safely manage itself.

!!! warning "Escalation"

    A token with `CreateAdminToken` or `UpdateAdminToken` in its scope can create a token with `*`, so it is equivalent to `*`.

The secret is only returned when the token is created. An imported token has no `secret_token`; to get a new secret, replace the resource.
