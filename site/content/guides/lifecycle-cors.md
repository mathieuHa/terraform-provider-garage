# Lifecycle and CORS rules

Expire temporary objects, abort stalled uploads and let a browser application upload directly to a bucket.

```terraform
--8<-- "examples/guides/lifecycle-cors/main.tf"
```

## Lifecycle rules

Each rule has an action and an optional filter:

- `expiration_days` deletes objects that many days after their creation; `expiration_date` deletes them from a date on.
- `abort_incomplete_multipart_upload_days` aborts multipart uploads that never completed, and frees the space their parts use.
- `filter` restricts the rule to a key prefix and to an object size range. Write the conditions side by side: when there are several, the provider wraps them in the `And` element that Garage requires.
- `enabled = false` keeps a rule without applying it.

Garage runs the rules once a day.

## CORS rules

A browser sends a preflight request before a cross-origin `PUT`. The rule above answers it for `https://app.example.com`, allows every request header, lets the application read the `ETag` of an upload, and lets the browser cache the answer for an hour.

## Who manages the rules

The two buckets show both modes:

- `uploads` sets `cors_rules` and `lifecycle_rules`, so the provider is **authoritative**: a rule added through the S3 API shows as drift and is removed on the next apply.
- `app-managed` sets neither, so the provider **leaves the rules alone**. Use this when an application manages its own rules through the S3 API with an `owner` key.

Set an attribute to `[]` to remove every rule and keep it managed.
