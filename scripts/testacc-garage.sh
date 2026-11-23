#!/usr/bin/env bash
set -euo pipefail

name="${GARAGE_TEST_CONTAINER:-garage-provider-acctest}"
image="${GARAGE_IMAGE:-dxflrs/garage:v2.5.0}"
s3_port="${GARAGE_TEST_S3_PORT:-13900}"
admin_port="${GARAGE_TEST_ADMIN_PORT:-13903}"
web_port="${GARAGE_TEST_WEB_PORT:-13902}"
token="${GARAGE_TEST_TOKEN:-acceptance-test-token}"
workdir="${TMPDIR:-/tmp}/${name}"

stop() {
  docker rm -f "$name" >/dev/null 2>&1 || true
  rm -rf "$workdir"
}

start() {
  stop
  mkdir -p "$workdir"
  cat > "$workdir/garage.toml" <<TOML
metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "[::]:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "$(openssl rand -hex 32)"

[s3_api]
s3_region = "garage"
api_bind_addr = "[::]:3900"
root_domain = ".s3.garage.localhost"

[s3_web]
bind_addr = "[::]:3902"
root_domain = ".web.garage.localhost"

[admin]
api_bind_addr = "[::]:3903"
admin_token = "${token}"
TOML

  docker run -d --name "$name" \
    -p "127.0.0.1:${s3_port}:3900" -p "127.0.0.1:${web_port}:3902" -p "127.0.0.1:${admin_port}:3903" \
    -v "$workdir/garage.toml:/etc/garage.toml:ro" \
    "$image" /garage server >/dev/null

  for _ in $(seq 1 60); do
    curl -fsS "http://127.0.0.1:${admin_port}/health" >/dev/null 2>&1 && break
    curl -fsS -o /dev/null -H "Authorization: Bearer ${token}" "http://127.0.0.1:${admin_port}/v2/GetClusterStatus" 2>/dev/null && break
    sleep 1
  done

  node_id="$(curl -fsS -H "Authorization: Bearer ${token}" "http://127.0.0.1:${admin_port}/v2/GetClusterStatus" | jq -r '.nodes[0].id')"
  docker exec "$name" /garage layout assign -z dc1 -c 1G "$node_id" >/dev/null 2>&1
  docker exec "$name" /garage layout apply --version 1 >/dev/null 2>&1

  for _ in $(seq 1 60); do
    [[ "$(curl -fsS "http://127.0.0.1:${admin_port}/health" -o /dev/null -w '%{http_code}' 2>/dev/null)" == "200" ]] && break
    sleep 1
  done

  echo "export GARAGE_ENDPOINT=http://127.0.0.1:${admin_port}"
  echo "export GARAGE_TOKEN=${token}"
  echo "export GARAGE_TEST_NODE_ID=${node_id}"
  echo "export GARAGE_TEST_S3_ENDPOINT=http://127.0.0.1:${s3_port}"
  echo "export GARAGE_TEST_WEB_ENDPOINT=http://127.0.0.1:${web_port}"
}

case "${1:-start}" in
  start) start ;;
  stop) stop ;;
  *) echo "usage: $0 [start|stop]" >&2; exit 2 ;;
esac
