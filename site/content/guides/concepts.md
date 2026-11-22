# Concepts

Garage implements the S3 API but not IAM. If you come from AWS, MinIO or Ceph RGW, access works differently, and the resources follow Garage's model.

## Buckets and aliases

A bucket has an ID of 64 hexadecimal characters and is reached through **aliases**:

- a **global alias** is a bucket name visible to every key. `garage_bucket.global_alias` manages one; add others with `garage_bucket_alias`.
- a **local alias** is a bucket name visible to a single key, set with `garage_bucket_alias` and `access_key_id`. Two keys can use the same local name for different buckets.

Garage refuses to remove the last alias of a bucket, and to delete a bucket that still contains objects.

## Keys and permissions

An **access key** is the identity: there are no users. A key has an ID (`GK` and 24 hexadecimal characters), a secret, an optional expiration and an optional right to create buckets.

A key gets permissions on whole buckets with `garage_bucket_permission`:

| Permission | Allows |
|---|---|
| `read` | Read objects and list the bucket |
| `write` | Write and delete objects |
| `owner` | Change the bucket through the S3 API (website, CORS, lifecycle) and delete it |

There are no policies, conditions or prefix-scoped grants. To separate data, use separate buckets.

## Public access

There is no public-read bucket policy. To serve a bucket anonymously, enable website hosting on it: Garage serves it on its web endpoint (port `3902` by default), using the `root_domain` of the `[s3_web]` section. See [Static website](website.md).

## What the provider manages

| Setting | Behaviour |
|---|---|
| Quotas (`max_size`, `max_objects`) and `website` | Authoritative: removing them from the configuration removes the quota or disables website hosting |
| `cors_rules`, `lifecycle_rules` | Authoritative only when set. When absent, rules set through the S3 API are left alone; `[]` removes every rule |
| `garage_bucket_permission` | Authoritative for the key and bucket pair; destroying it revokes the key's access to the bucket |
| `garage_cluster_layout` | Authoritative for the node roles; destroying it leaves the layout unchanged |

## Coming from S3 and IAM

| AWS or Ceph RGW | Garage provider |
|---|---|
| `aws_s3_bucket` | `garage_bucket` |
| `aws_iam_user` and `aws_iam_access_key` | `garage_key` |
| `aws_iam_user_policy` | `garage_bucket_permission` |
| `aws_s3_bucket_lifecycle_configuration` | `garage_bucket.lifecycle_rules` |
| `aws_s3_bucket_cors_configuration` | `garage_bucket.cors_rules` |
| `aws_s3_bucket_website_configuration` | `garage_bucket.website` |
| Bucket quotas outside Terraform (`radosgw-admin quota set`) | `garage_bucket.max_size`, `max_objects` |
