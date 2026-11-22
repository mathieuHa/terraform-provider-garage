# Static website

Serve a bucket as a website on the Garage web endpoint, with redirect rules, and give a deployment key write access to it.

```terraform
--8<-- "examples/guides/website/main.tf"
```

- With `website`, Garage serves the bucket anonymously on its web endpoint. A request for `https://www.example.com/` is answered from the bucket whose alias matches the host name. Garage also accepts `<bucket>.<root_domain>`, where `root_domain` comes from the `[s3_web]` section of `garage.toml`.
- `routing_rules` are evaluated in order. The first redirects `blog/…` to `posts/…` with a permanent redirect. The second answers missing pages under `docs/` with `docs/index.html`. A rule without `http_redirect_code` uses `302`.
- `www-deploy` is a key for the CI job that uploads the site, for example with `aws s3 sync`.

Removing `website` from the configuration disables website hosting.

!!! tip "TLS"

    The Garage web endpoint speaks plain HTTP. Put a reverse proxy in front of it for HTTPS, and route the site's host names to it.
