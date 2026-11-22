# Quickstart

This walks you from an empty directory to a bucket, an access key and its permission on your Garage cluster, in about five minutes.

## Requirements

- [OpenTofu](https://opentofu.org/docs/intro/install/) 1.6 or later, or Terraform 1.5 or later.
- Garage 2.3.0 or later. Earlier 2.x releases have no CORS or lifecycle support in the admin API, which the bucket resource needs.
- Network access to the admin API of one node, port `3903` by default.

## 1. Get an admin token

The provider authenticates with a bearer token. To start, use the `admin_token` from the `[admin]` section of `garage.toml`, or the value of `GARAGE_ADMIN_TOKEN` if the daemon reads it from the environment.

That token can do everything. Once the provider works, replace it with a token restricted to what OpenTofu needs: see [Least-privilege admin token](guides/admin-token.md).

## 2. Configure the provider

The provider reads its settings from the environment, so nothing secret ends up in your configuration:

```bash
export GARAGE_ENDPOINT="http://garage.example.com:3903"
export GARAGE_TOKEN="<admin token>"
```

For an admin API behind TLS with a private CA, also set `GARAGE_CA_CERT_FILE`. All settings can also be set in the `provider` block; see the [provider configuration](reference/provider.md).

## 3. Write the configuration

Create `main.tf`:

```terraform
--8<-- "examples/guides/quickstart/main.tf"
```

It creates:

- a bucket `photos` limited to 10 GiB,
- an access key `photos-app`,
- read and write permission for that key on that bucket.

## 4. Apply

```bash
tofu init
tofu apply
```

The bucket is usable as soon as the apply finishes. Try it with any S3 client, using the key from the outputs and the `s3_region` of your `garage.toml` (`garage` by default):

```bash
export AWS_ACCESS_KEY_ID="$(tofu output -raw access_key_id)"
export AWS_SECRET_ACCESS_KEY="$(tofu output -raw secret_access_key)"
export AWS_DEFAULT_REGION=garage
echo hello | aws --endpoint-url http://garage.example.com:3900 s3 cp - s3://photos/hello.txt
```

Run `tofu plan` again and it reports no changes.

## 5. Clean up

```bash
aws --endpoint-url http://garage.example.com:3900 s3 rm s3://photos/hello.txt
tofu destroy
```

Garage refuses to delete a bucket that still contains objects, so empty it first.

## Next steps

- [Concepts](guides/concepts.md): buckets, aliases, keys and permissions in Garage.
- [Application credentials in OpenBao or Vault](guides/app-credentials.md): one key per application, published where the application reads it.
- [Supported resources](resources.md): everything the provider can manage.
