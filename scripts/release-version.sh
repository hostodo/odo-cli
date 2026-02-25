#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/release-version.sh [major|minor|patch] [--push] [--dry-run]

Examples:
  ./scripts/release-version.sh
  ./scripts/release-version.sh minor --push
  ./scripts/release-version.sh major --dry-run

Notes:
  - Default bump type is patch.
  - Creates an annotated git tag like v1.2.3.
  - Use --push to push the new tag to origin (triggers GitHub release workflow).
EOF
}

BUMP_TYPE="patch"
PUSH_TAG=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    major|minor|patch)
      BUMP_TYPE="$1"
      shift
      ;;
    --push)
      PUSH_TAG=true
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "Error: not inside a git repository." >&2
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  if [[ "${DRY_RUN}" == true ]]; then
    echo "Warning: git working tree is dirty (allowed in dry-run mode)." >&2
  else
    echo "Error: git working tree is dirty. Commit or stash changes first." >&2
    exit 1
  fi
fi

LATEST_TAG="$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | head -n 1)"
if [[ -z "${LATEST_TAG}" ]]; then
  LATEST_TAG="v0.0.0"
fi

if [[ ! "${LATEST_TAG}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "Error: latest tag '${LATEST_TAG}' is not valid semver (vMAJOR.MINOR.PATCH)." >&2
  exit 1
fi

MAJOR="${BASH_REMATCH[1]}"
MINOR="${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"

case "${BUMP_TYPE}" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEXT_TAG="v${MAJOR}.${MINOR}.${PATCH}"

if git rev-parse "${NEXT_TAG}" >/dev/null 2>&1; then
  echo "Error: tag ${NEXT_TAG} already exists." >&2
  exit 1
fi

echo "Latest tag: ${LATEST_TAG}"
echo "Bump type : ${BUMP_TYPE}"
echo "Next tag  : ${NEXT_TAG}"

if [[ "${DRY_RUN}" == true ]]; then
  echo "Dry run enabled. No tag created."
  exit 0
fi

git tag -a "${NEXT_TAG}" -m "Release ${NEXT_TAG}"
echo "Created tag ${NEXT_TAG}"

if [[ "${PUSH_TAG}" == true ]]; then
  git push origin "${NEXT_TAG}"
  echo "Pushed ${NEXT_TAG} to origin."
  echo "GitHub Actions release workflow should now run."
else
  echo "Tag not pushed. Push when ready:"
  echo "  git push origin ${NEXT_TAG}"
fi
