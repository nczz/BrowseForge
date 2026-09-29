#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

version="${1:-}"
if [[ -z "$version" ]]; then
  version="$(python3 - <<'PY'
import pathlib, re, sys
for path in (pathlib.Path("internal/runtime/support.go"), pathlib.Path("internal/browser/download.go")):
    match = re.search(r'BrowseForgeChromiumVersion\s*=\s*"([^"]+)"', path.read_text())
    if match:
        print(match.group(1))
        sys.exit(0)
PY
)"
fi

if [[ -z "$version" ]]; then
  echo "Could not read BrowseForgeChromiumVersion from internal/runtime/support.go" >&2
  exit 1
fi

base_url="${BROWSEFORGE_CHROMIUM_RELEASE_BASE_URL:-https://github.com/nczz/browseforge-runtime-chromium/releases/download}"
base_url="${base_url%/}"

missing=0
checksums_url="${base_url}/${version}/checksums.txt"
echo "Checking ${checksums_url}"
checksums=""
if ! checksums="$(curl -fsSL "$checksums_url" 2>/dev/null)"; then
  echo "BrowseForge Chromium runtime metadata unavailable: ${checksums_url}" >&2
  missing=1
fi

manifest_url="${base_url}/${version}/runtime.manifest.json"
echo "Checking ${manifest_url}"
manifest_json="$(curl -fsSL "${manifest_url}" 2>/dev/null)" || {
  echo "BrowseForge Chromium runtime metadata unavailable: ${manifest_url}" >&2
  missing=1
}
if [[ -n "${manifest_json}" ]]; then
  manifest_version="$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("version", ""))' <<<"${manifest_json}")"
  if [[ "${manifest_version}" != "${version}" ]]; then
    echo "BrowseForge Chromium runtime manifest version mismatch: got ${manifest_version:-<empty>}, want ${version}" >&2
    missing=1
  fi
fi

for metadata in runtime.manifest.json detector-summary.json; do
  if [[ -n "$checksums" && "$checksums" != *"$metadata"* ]]; then
    echo "BrowseForge Chromium runtime checksum entry missing: ${metadata}" >&2
    missing=1
  fi
done

for suffix in linux-x64 linux-arm64 macos-arm64 macos-x64 windows-x64; do
  filename="browseforge-runtime-chromium-${version}-${suffix}.zip"
  url="${base_url}/${version}/${filename}"
  echo "Checking ${url}"
  if ! curl -fsI "$url" >/dev/null; then
    echo "BrowseForge Chromium runtime artifact unavailable: ${url}" >&2
    missing=1
  fi
  if [[ -n "$checksums" && "$checksums" != *"$filename"* ]]; then
    echo "BrowseForge Chromium runtime checksum entry missing: ${filename}" >&2
    missing=1
  fi
done

exit "$missing"
