#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT_DIR/VERSION"
SEMVER_RE='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'

usage() {
  cat <<'USAGE'
Usage:
  scripts/semver.sh show
  scripts/semver.sh set <version>
  scripts/semver.sh bump <major|minor|patch>
  scripts/semver.sh tag [--push]

Notes:
  - VERSION is stored without leading 'v' (example: 1.2.3).
  - tag creates an annotated git tag in form v<version>.
USAGE
}

read_version() {
  tr -d '[:space:]' < "$VERSION_FILE"
}

validate_semver() {
  local v="$1"
  [[ "$v" =~ $SEMVER_RE ]]
}

write_version() {
  local v="$1"
  printf '%s\n' "$v" > "$VERSION_FILE"
}

bump_version() {
  local current="$1"
  local part="$2"
  local base="${current%%[-+]*}"
  local major minor patch

  IFS='.' read -r major minor patch <<< "$base"
  case "$part" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
    *)
      echo "invalid bump part: $part" >&2
      exit 1
      ;;
  esac
  printf '%d.%d.%d\n' "$major" "$minor" "$patch"
}

ensure_clean_worktree() {
  if ! git -C "$ROOT_DIR" diff --quiet || ! git -C "$ROOT_DIR" diff --cached --quiet; then
    echo "worktree must be clean before tagging" >&2
    exit 1
  fi
}

cmd="${1:-}"
case "$cmd" in
  show)
    read_version
    ;;
  set)
    v="${2:-}"
    if [[ -z "$v" ]]; then
      usage
      exit 1
    fi
    if ! validate_semver "$v"; then
      echo "invalid semver: $v" >&2
      exit 1
    fi
    write_version "$v"
    echo "$v"
    ;;
  bump)
    part="${2:-}"
    if [[ -z "$part" ]]; then
      usage
      exit 1
    fi
    current="$(read_version)"
    if ! validate_semver "$current"; then
      echo "VERSION is not valid semver: $current" >&2
      exit 1
    fi
    next="$(bump_version "$current" "$part")"
    write_version "$next"
    echo "$next"
    ;;
  tag)
    push_flag="${2:-}"
    current="$(read_version)"
    if ! validate_semver "$current"; then
      echo "VERSION is not valid semver: $current" >&2
      exit 1
    fi
    ensure_clean_worktree

    tag="v$current"
    if git -C "$ROOT_DIR" rev-parse --verify --quiet "$tag" >/dev/null; then
      echo "tag already exists: $tag" >&2
      exit 1
    fi

    git -C "$ROOT_DIR" tag -a "$tag" -m "Release $tag"
    echo "created tag: $tag"

    if [[ "$push_flag" == "--push" ]]; then
      git -C "$ROOT_DIR" push origin "$tag"
      echo "pushed tag: $tag"
    elif [[ -n "$push_flag" ]]; then
      usage
      exit 1
    fi
    ;;
  *)
    usage
    exit 1
    ;;
esac
