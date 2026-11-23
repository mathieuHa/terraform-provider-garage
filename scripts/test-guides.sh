#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
: "${GARAGE_ENDPOINT:?start a Garage with scripts/testacc-garage.sh first}"
: "${GARAGE_TOKEN:?start a Garage with scripts/testacc-garage.sh first}"
cli="${TF_CLI:-tofu}"
bao_name="${GUIDES_BAO_CONTAINER:-garage-provider-guides-bao}"
bao_port="${GUIDES_BAO_PORT:-18200}"
guides=(quickstart app-credentials website lifecycle-cors admin-token import cluster-layout)

work="$(mktemp -d)"
cleanup() {
  docker rm -f "$bao_name" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT

for dir in "$root"/examples/guides/*/; do
  name="$(basename "$dir")"
  if [[ ! " ${guides[*]} " =~ " ${name} " ]]; then
    echo "examples/guides/${name} is not tested: add it to scripts/test-guides.sh" >&2
    exit 1
  fi
done

(cd "$root" && go build -o "$work/bin/terraform-provider-garage" .)
cat > "$work/cli.rc" <<RC
provider_installation {
  dev_overrides {
    "registry.opentofu.org/mathieuha/garage"  = "$work/bin"
    "registry.terraform.io/mathieuha/garage"  = "$work/bin"
  }
  direct {}
}
RC
export TF_CLI_CONFIG_FILE="$work/cli.rc"

docker rm -f "$bao_name" >/dev/null 2>&1 || true
docker run -d --name "$bao_name" -p "127.0.0.1:${bao_port}:8200" \
  -e BAO_DEV_ROOT_TOKEN_ID=guides openbao/openbao:2.7.3 \
  server -dev -dev-listen-address=0.0.0.0:8200 >/dev/null
export VAULT_ADDR="http://127.0.0.1:${bao_port}" VAULT_TOKEN=guides
for _ in $(seq 1 30); do
  curl -fsS "$VAULT_ADDR/v1/sys/health" >/dev/null 2>&1 && break
  sleep 1
done

admin() {
  curl -fsS -H "Authorization: Bearer ${GARAGE_TOKEN}" -H 'Content-Type: application/json' "$@"
}

run() {
  local name="$1" log="$work/$1.log" dir="$work/$1"
  local args=()
  mkdir -p "$dir"
  cp "$root/examples/guides/$name/main.tf" "$dir/"
  if ! grep -q required_providers "$dir/main.tf"; then
    cat > "$dir/providers.tf" <<'TF'
terraform {
  required_providers {
    garage = { source = "mathieuha/garage" }
    vault  = { source = "hashicorp/vault", version = "4.8.0" }
  }
}
TF
  fi

  if [[ "$name" == import ]]; then
    local bucket key
    bucket="$(admin -d '{"globalAlias":"legacy-data"}' "$GARAGE_ENDPOINT/v2/CreateBucket" | jq -r .id)"
    key="$(admin -d '{"name":"legacy-app"}' "$GARAGE_ENDPOINT/v2/CreateKey" | jq -r .accessKeyId)"
    admin -o /dev/null -d "{\"bucketId\":\"$bucket\",\"accessKeyId\":\"$key\",\"permissions\":{\"read\":true,\"write\":true}}" \
      "$GARAGE_ENDPOINT/v2/AllowBucketKey"
    args=(-var "legacy_key_id=$key")
  fi

  cd "$dir"
  "$cli" init -input=false -no-color >>"$log" 2>&1 || return 1
  if [[ "$name" == import ]]; then
    "$cli" plan -input=false -no-color "${args[@]}" >>"$log" 2>&1 || return 1
    grep -q 'Plan: 3 to import, 0 to add, 0 to change, 0 to destroy.' "$log" || { echo "expected a pure import" >>"$log"; return 1; }
  fi
  "$cli" apply -auto-approve -input=false -no-color "${args[@]}" >>"$log" 2>&1 || return 1
  "$cli" plan -detailed-exitcode -input=false -no-color "${args[@]}" >>"$log" 2>&1 || { echo "plan after apply is not empty" >>"$log"; return 1; }
  "$cli" destroy -auto-approve -input=false -no-color "${args[@]}" >>"$log" 2>&1 || return 1
}

failed=0
for name in "${guides[@]}"; do
  if (run "$name"); then
    echo "PASS  $name"
  else
    echo "FAIL  $name"
    sed 's/^/      /' "$work/$name.log" | tail -40
    failed=1
  fi
done
exit "$failed"
