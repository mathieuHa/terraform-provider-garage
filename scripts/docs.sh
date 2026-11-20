#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cd "$root"
go build -o "$work/bin/terraform-provider-garage" .

mkdir -p "$work/schema"
cat > "$work/schema/main.tf" <<'TF'
terraform {
  required_providers {
    garage = { source = "mathieuha/garage" }
  }
}
TF
cat > "$work/tofurc" <<RC
provider_installation {
  dev_overrides { "registry.opentofu.org/mathieuha/garage" = "$work/bin" }
  direct {}
}
RC

(cd "$work/schema" && TF_CLI_CONFIG_FILE="$work/tofurc" tofu providers schema -json 2>/dev/null) |
  jq '.provider_schemas |= with_entries(.key = "registry.terraform.io/hashicorp/garage")' > "$work/schema.json"

go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate \
  --provider-name garage --providers-schema "$work/schema.json"
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 validate --provider-name garage
