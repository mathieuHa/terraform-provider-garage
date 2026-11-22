# API coverage

How the provider uses each endpoint of the Garage admin API v2, as of Garage 2.5.

## Covered

| Endpoint | Used by |
|---|---|
| `CreateBucket`, `GetBucketInfo`, `UpdateBucket`, `DeleteBucket` | `garage_bucket` (quotas, website and routing rules, CORS rules, lifecycle rules), `garage_bucket` data source |
| `ListBuckets` | `garage_buckets` data source |
| `AddBucketAlias`, `RemoveBucketAlias` | `garage_bucket` (its global alias), `garage_bucket_alias` (other global aliases, local aliases) |
| `AllowBucketKey`, `DenyBucketKey` | `garage_bucket_permission` |
| `CreateKey`, `ImportKey`, `GetKeyInfo`, `UpdateKey`, `DeleteKey` | `garage_key`, `garage_key` data source |
| `ListKeys` | `garage_keys` data source |
| `CreateAdminToken`, `GetAdminTokenInfo`, `UpdateAdminToken`, `DeleteAdminToken` | `garage_admin_token`, `garage_admin_token` data source |
| `GetCurrentAdminTokenInfo` | `garage_admin_token` data source without arguments |
| `ListAdminTokens` | Acceptance tests only |
| `GetClusterLayout`, `UpdateClusterLayout`, `ApplyClusterLayout`, `RevertClusterLayout` | `garage_cluster_layout`, `garage_cluster_layout` data source |
| `GetClusterStatus` | `garage_cluster_status` data source |
| `GetClusterHealth` | `garage_cluster_health` data source |
| `GetNodeInfo` | `garage_node` data source |

## Not covered yet

| Endpoint | Possible use |
|---|---|
| `GetWorkerVariable`, `SetWorkerVariable` | A resource for background worker tuning, such as resync and scrub tranquility |
| `GetClusterStatistics`, `GetNodeStatistics` | Data sources. Their main content is a free-form text that Garage does not guarantee to keep stable |
| `GetClusterLayoutHistory` | A data source for layout versions and their synchronisation |
| `PreviewClusterLayoutChanges` | Showing the partition moves of a layout change before it is applied |
| `ListBucketsInfo`, `ListKeysInfo` | Paginated lists with full details, added in Garage 2.5 |

## Out of scope

These endpoints are one-off operations, not configuration, so they have no place in a declarative plan. Use the `garage` CLI or call the admin API directly.

| Endpoint | Operation |
|---|---|
| `ConnectClusterNodes` | Connect a node to others when building a cluster |
| `ClusterLayoutSkipDeadNodes` | Force a layout change past nodes that are down |
| `CleanupIncompleteUploads` | Purge incomplete multipart uploads (use `abort_incomplete_multipart_upload_days` in a lifecycle rule to do it continuously) |
| `InspectObject` | Inspect the internal representation of an object |
| `CreateMetadataSnapshot` | Snapshot the metadata database |
| `LaunchRepairOperation` | Start a repair, scrub or rebalance |
| `ListWorkers`, `GetWorkerInfo` | Background worker status |
| `GetBlockInfo`, `ListBlockErrors`, `RetryBlockResync`, `PurgeBlocks` | Data block maintenance |

S3 operations on objects are not part of the admin API. Manage objects with an S3 client, or with the `aws_s3_object` resource of the AWS provider pointed at the Garage S3 endpoint.
