#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release-push.sh vX.Y.Z

Runs release preflight, creates an annotated tag, and pushes only that tag.

Environment:
  SKIP_PREFLIGHT=1   Do not run scripts/release-preflight.sh first.
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

[[ "$(git status --short)" == "" ]] || die "working tree must be clean before tagging"
major="${version#v}"
major="${major%%.*}"
case "$major" in
  1) expected_branch="v1" ;;
  2) expected_branch="main" ;;
  *) die "unsupported release major: v${major}" ;;
esac
[[ "$(git branch --show-current)" == "$expected_branch" ]] || die "${version} release tags must be created from ${expected_branch}"

if [[ "${SKIP_PREFLIGHT:-}" != "1" ]]; then
  run scripts/release-preflight.sh "$version"
else
  echo "warning: skipping release preflight because SKIP_PREFLIGHT=1"
fi

if git rev-parse "$version" >/dev/null 2>&1; then
  die "local tag already exists: $version"
fi

if git ls-remote --exit-code --tags origin "refs/tags/${version}" >/dev/null 2>&1; then
  die "remote tag already exists: $version"
fi

run git tag -a "$version" -m "Release ${version}"
run git push origin "$version"

echo "release tag pushed: ${version}"
echo "watch the release workflow with: gh run watch"
