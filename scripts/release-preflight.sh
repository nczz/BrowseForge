#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release-preflight.sh vX.Y.Z

Environment:
  CAMOUFOX_PATH=/path/to/camoufox         Override Camoufox binary path.
  CLOAKBROWSER_PATH=/path/to/chromium     Override CloakBrowser binary path.
  REQUIRE_CLOAKBROWSER=1                  Fail if the CloakBrowser runtime spike cannot run.
  SKIP_CAMOUFOX=1                         Skip the Camoufox runtime spike.
  SKIP_CLOAKBROWSER=1                     Skip the CloakBrowser runtime spike.
  SKIP_DOCKER=1                           Skip the linux/amd64 Docker build.
USAGE
}

die() {
  echo "error: $*" >&2
  exit 1
}

run() {
  echo "+ $*"
  "$@"
}

version="${1:-}"
if [[ -z "$version" || "${version}" == "-h" || "${version}" == "--help" ]]; then
  usage
  exit 1
fi

[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "version must match vX.Y.Z, got: $version"

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"
tmpdir="$(mktemp -d)"
asset_server_pid=""

cleanup() {
  if [[ -n "$asset_server_pid" ]]; then
    kill "$asset_server_pid" >/dev/null 2>&1 || true
    wait "$asset_server_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmpdir"
}
trap cleanup EXIT

[[ "$(git status --short)" == "" ]] || die "working tree must be clean before release"

current_branch="$(git branch --show-current)"
major="${version#v}"
major="${major%%.*}"
case "$major" in
  1) expected_branch="v1" ;;
  2) expected_branch="main" ;;
  *) die "unsupported release major: v${major}" ;;
esac
[[ "$current_branch" == "$expected_branch" ]] || die "${version} release must be prepared from ${expected_branch}, current branch: $current_branch"

if git rev-parse "$version" >/dev/null 2>&1; then
  die "local tag already exists: $version"
fi

head_sha="$(git rev-parse HEAD)"
origin_sha="$(git rev-parse "origin/${expected_branch}" 2>/dev/null || true)"
if [[ -n "$origin_sha" && "$head_sha" != "$origin_sha" ]]; then
  die "HEAD ($head_sha) does not match origin/${expected_branch} ($origin_sha); push ${expected_branch} first"
fi

command -v go >/dev/null || die "go is required"
command -v node >/dev/null || die "node is required"
command -v docker >/dev/null || die "docker is required"
command -v rg >/dev/null || die "ripgrep (rg) is required"

if ! rg -q "BROWSEFORGE_VERSION:-${version}" docker/docker-compose.yml; then
  die "docker/docker-compose.yml default BROWSEFORGE_VERSION is not ${version}"
fi

if ! rg -q "ghcr.io/nczz/browseforge:${version}" docker/README.md docs/linux-server.md; then
  die "Docker docs must reference ghcr.io/nczz/browseforge:${version}"
fi

run go build -ldflags "-s -w -X main.Version=${version#v}" -o "$tmpdir/BrowseForge" ./cmd/server
"$tmpdir/BrowseForge" doctor | rg -q "BrowseForge ${version#v}" || die "release binary version does not match ${version}"

go_packages="$(go list ./... | rg -v '/internal/spike$')"
run go test -count=1 $go_packages
run go vet ./...
run node --check extension/sidebar/app.js
run node -e "JSON.parse(require('fs').readFileSync('extension/manifest.json','utf8')); JSON.parse(require('fs').readFileSync('extension/_locales/en/messages.json','utf8')); JSON.parse(require('fs').readFileSync('extension/_locales/zh_TW/messages.json','utf8')); console.log('extension json ok')"
run node -e "const fs=require('fs'); const html=fs.readFileSync('internal/api/dashboard.html','utf8'); const m=html.match(/<script>([\\s\\S]*)<\\/script>/); if(!m) throw new Error('script not found'); new Function(m[1]); console.log('dashboard js ok')"
run node scripts/check-i18n.js
run node scripts/check-doc-language.js

if command -v ruby >/dev/null; then
  run ruby -e "require 'yaml'; YAML.load_file('.github/workflows/release.yml'); puts 'workflow yaml ok'"
else
  echo "warning: ruby not found; skipping workflow YAML parse"
fi

run docker compose -f docker/docker-compose.yml config

if [[ "${SKIP_CAMOUFOX:-}" != "1" ]]; then
  camoufox_path="${CAMOUFOX_PATH:-$repo_root/browsers/camoufox/Camoufox.app/Contents/MacOS/camoufox}"
  [[ -x "$camoufox_path" ]] || die "Camoufox binary not found or not executable: $camoufox_path"
  run env CAMOUFOX_PATH="$camoufox_path" go test -count=1 -run '^TestPlaywrightBindEndpointWithCamoufox$' -v ./internal/spike
else
  echo "warning: skipping Camoufox runtime spike because SKIP_CAMOUFOX=1"
fi

if [[ "${SKIP_CLOAKBROWSER:-}" != "1" ]]; then
  cloakbrowser_path="${CLOAKBROWSER_PATH:-$repo_root/browsers/cloakbrowser/Chromium.app/Contents/MacOS/Chromium}"
  if [[ -x "$cloakbrowser_path" ]]; then
    run env CLOAKBROWSER_SPIKE=1 CLOAKBROWSER_PATH="$cloakbrowser_path" go test -count=1 -timeout 45s -run '^TestPlaywrightBindEndpointWithCloakBrowser$' -v ./internal/spike
  elif [[ "${REQUIRE_CLOAKBROWSER:-}" == "1" ]]; then
    die "CloakBrowser binary not found or not executable: $cloakbrowser_path"
  else
    echo "warning: skipping CloakBrowser runtime spike; set CLOAKBROWSER_PATH or REQUIRE_CLOAKBROWSER=1 to enforce it"
  fi
else
  echo "warning: skipping CloakBrowser runtime spike because SKIP_CLOAKBROWSER=1"
fi

if [[ "${SKIP_DOCKER:-}" != "1" ]]; then
  command -v zip >/dev/null || die "zip is required"
  command -v python3 >/dev/null || die "python3 is required"

  asset_root="$tmpdir/release-assets"
  asset_version_dir="$asset_root/$version"
  package_dir="$tmpdir/package/BrowseForge-lite"
  mkdir -p "$asset_version_dir" "$package_dir/data" "$package_dir/profiles" "$package_dir/logs" "$package_dir/examples"

  run env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=${version#v}" \
    -o "$package_dir/BrowseForge" \
    ./cmd/server
  run cp -R extension "$package_dir/extension"
  run cp data/fingerprints-chrome-macos.json data/fingerprints-chrome-windows.json data/fingerprints-firefox-macos.json data/fingerprints-firefox-windows.json "$package_dir/data/"
  if [[ -d examples ]]; then
    cp -R examples/. "$package_dir/examples/"
  fi
  run cp config.default.json "$package_dir/config.json"
  run cp README.md README.zh-TW.md API.md API.zh-TW.md "$package_dir/"
  (
    cd "$tmpdir/package"
    run zip -qr "$asset_version_dir/BrowseForge-${version}-lite-linux-x64.zip" BrowseForge-lite
  )

  port_file="$tmpdir/asset-server.port"
  python3 - "$asset_root" "$port_file" <<'PY' &
import functools
import http.server
import sys

root = sys.argv[1]
port_file = sys.argv[2]

handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=root)
server = http.server.ThreadingHTTPServer(("0.0.0.0", 0), handler)
with open(port_file, "w", encoding="utf-8") as fh:
    fh.write(str(server.server_port))
server.serve_forever()
PY
  asset_server_pid="$!"
  for _ in {1..100}; do
    [[ -s "$port_file" ]] && break
    sleep 0.1
  done
  [[ -s "$port_file" ]] || die "local release asset server did not start"
  asset_server_port="$(cat "$port_file")"
  asset_base_url="http://host.docker.internal:${asset_server_port}"

  run docker build \
    --add-host=host.docker.internal:host-gateway \
    --platform linux/amd64 \
    -f docker/Dockerfile.run \
    --build-arg "BROWSEFORGE_VERSION=${version}" \
    --build-arg BROWSEFORGE_ARCH=linux-x64 \
    --build-arg "BROWSEFORGE_RELEASE_BASE_URL=${asset_base_url}" \
    -t "browseforge:verify-${version}" \
    docker
  run docker run --rm --platform linux/amd64 --entrypoint /bin/bash "browseforge:verify-${version}" -n /entrypoint.sh
else
  echo "warning: skipping Docker build because SKIP_DOCKER=1"
fi

echo "release preflight passed for ${version}"
