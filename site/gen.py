import json
import os
import re
import shutil
import sys
import tomllib

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SITE = os.path.join(ROOT, "site")
BUILD = os.path.join(SITE, "build")
DOCS = os.path.join(BUILD, "docs")

AREAS = {
    "bucket": "Buckets",
    "buckets": "Buckets",
    "bucket_alias": "Buckets",
    "key": "Keys and permissions",
    "keys": "Keys and permissions",
    "bucket_permission": "Keys and permissions",
    "admin_token": "Admin tokens",
    "cluster_layout": "Cluster",
    "cluster_health": "Cluster",
    "cluster_status": "Cluster",
    "node": "Cluster",
}
AREA_ORDER = ["Buckets", "Keys and permissions", "Admin tokens", "Cluster"]
GUIDE_ORDER = [
    "guides/concepts.md",
    "guides/app-credentials.md",
    "guides/website.md",
    "guides/lifecycle-cors.md",
    "guides/admin-token.md",
    "guides/cluster-layout.md",
    "guides/import.md",
]

KINDS = (("resources", "Resource"), ("data-sources", "Data source"))
CALLOUTS = {"~>": "warning", "->": "note", "!>": "danger"}


def area_of(name):
    if name not in AREAS:
        raise SystemExit(f"no area for {name}; add it to AREAS in site/gen.py")
    return AREAS[name]


def split_frontmatter(text):
    if text.startswith("---\n"):
        end = text.index("\n---\n", 4)
        return yaml.safe_load(text[4:end]) or {}, text[end + 5:]
    return {}, text


def convert(body):
    out, lines, i = [], body.split("\n"), 0
    while i < len(lines):
        line = lines[i]
        m = re.match(r"^(~>|->|!>) (.*)$", line)
        if m:
            out.append(f"!!! {CALLOUTS[m.group(1)]}")
            out.append("")
            out.append("    " + m.group(2))
            i += 1
            while i < len(lines) and lines[i].strip():
                out.append("    " + lines[i])
                i += 1
            continue
        out.append(line)
        i += 1
    text = "\n".join(out)
    text = re.sub(r"\]\((\.\./(?:resources|data-sources)/[a-z0-9_]+)\)", r"](\1.md)", text)
    text = re.sub(r"\]\(([a-z0-9_]+)\)", r"](\1.md)", text)
    return text


def first_sentence(desc):
    desc = re.sub(r"\s+", " ", (desc or "").strip())
    m = re.match(r"(.+?\.)(\s|$)", desc)
    return (m.group(1) if m else desc).replace("|", "\\|")


def collect():
    items = []
    for kind, label in KINDS:
        src = os.path.join(ROOT, "docs", kind)
        for f in sorted(os.listdir(src)):
            if not f.endswith(".md"):
                continue
            name = f[:-3]
            meta, body = split_frontmatter(open(os.path.join(src, f)).read())
            items.append({"name": name, "kind": kind, "label": label, "area": area_of(name),
                          "desc": meta.get("description", ""), "body": body})
    return items


def catalogue(items, stats):
    names = {(it["kind"], it["name"]) for it in items}
    out = ["# Supported resources", "",
           f"The provider manages **{stats['resources']} resources** and reads **{stats['data_sources']} data sources** "
           "through the Garage admin API v2. See [API coverage](coverage.md) for how each admin endpoint is used.", ""]
    for area in AREA_ORDER:
        rows = [it for it in items if it["area"] == area]
        if not rows:
            continue
        out += [f"## {area}", "", "| Name | Kind | Description |", "|---|---|---|"]
        for it in sorted(rows, key=lambda x: ([k for k, _ in KINDS].index(x["kind"]), x["name"])):
            link = f"[`garage_{it['name']}`](reference/{it['kind']}/{it['name']}.md)"
            out.append(f"| {link} | {it['label']} | {first_sentence(it['desc'])} |")
        out.append("")
    return "\n".join(out)


def nav_reference(items):
    nav = []
    for area in AREA_ORDER:
        entries = []
        for it in sorted([x for x in items if x["area"] == area],
                         key=lambda x: ([k for k, _ in KINDS].index(x["kind"]), x["name"])):
            suffix = " (data source)" if it["kind"] == "data-sources" else ""
            entries.append({f"garage_{it['name']}{suffix}": f"reference/{it['kind']}/{it['name']}.md"})
        if entries:
            nav.append({area: entries})
    return nav


def toml_value(v):
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, (int, float)):
        return str(v)
    if isinstance(v, str):
        return json.dumps(v, ensure_ascii=False)
    if isinstance(v, list):
        return "[" + ", ".join(toml_value(x) for x in v) + "]"
    if isinstance(v, dict):
        return "{" + ", ".join(f"{json.dumps(k, ensure_ascii=False)} = {toml_value(x)}" for k, x in v.items()) + "}"
    raise TypeError(type(v))


def dump_toml(cfg):
    out = []
    for table, values in cfg.items():
        out.append(f"[{table}]")
        for k, v in values.items():
            if k == "nav":
                out.append("nav = [")
                out += [f"  {toml_value(x)}," for x in v]
                out.append("]")
            else:
                out.append(f"{json.dumps(k)} = {toml_value(v)}")
        out.append("")
    return "\n".join(out)


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    open(path, "w").write(text)


def main():
    if os.path.isdir(BUILD):
        shutil.rmtree(BUILD)
    items = collect()
    stats = {
        "resources": sum(1 for it in items if it["kind"] == "resources"),
        "data_sources": sum(1 for it in items if it["kind"] == "data-sources"),
    }
    for it in items:
        write(os.path.join(DOCS, "reference", it["kind"], it["name"] + ".md"), convert(it["body"]).lstrip("\n"))
    _, provider_body = split_frontmatter(open(os.path.join(ROOT, "docs", "index.md")).read())
    write(os.path.join(DOCS, "reference", "provider.md"), convert(provider_body).lstrip("\n"))
    write(os.path.join(DOCS, "resources.md"), catalogue(items, stats))

    content = os.path.join(SITE, "content")
    guides = []
    for dirpath, _, files in os.walk(content):
        for f in files:
            src = os.path.join(dirpath, f)
            rel = os.path.relpath(src, content)
            text = open(src).read()
            if f.endswith(".md"):
                for k, v in stats.items():
                    text = text.replace("{{ " + k + " }}", str(v))
                if rel.startswith("guides" + os.sep):
                    title = re.search(r"^# (.+)$", text, re.M).group(1)
                    guides.append((title, rel))
            write(os.path.join(DOCS, rel), text)
    missing = set(r for _, r in guides) - set(GUIDE_ORDER)
    if missing:
        raise SystemExit(f"guides missing from GUIDE_ORDER in site/gen.py: {sorted(missing)}")
    guides.sort(key=lambda g: GUIDE_ORDER.index(g[1]))

    cfg = tomllib.load(open(os.path.join(SITE, "zensical.base.toml"), "rb"))
    project = cfg["project"]
    project["docs_dir"] = "docs"
    project["site_dir"] = "public"
    project["markdown_extensions"]["pymdownx"]["snippets"]["base_path"] = [ROOT]
    project["nav"] = [
        {"Home": "index.md"},
        {"Quickstart": "quickstart.md"},
        {"Guides": [{t: r} for t, r in guides]},
        {"Supported resources": "resources.md"},
        {"API coverage": "coverage.md"},
        {"Reference": [{"Provider configuration": "reference/provider.md"}] + nav_reference(items)},
    ]
    write(os.path.join(BUILD, "zensical.toml"), dump_toml(cfg))
    json.dump(stats, sys.stdout)
    print()


if __name__ == "__main__":
    main()
