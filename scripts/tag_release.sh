#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/tag_release.sh <version> [message]

Creates an annotated Git tag (prefixed with v when missing), ensures the working
copy is clean, fast-forwards the current branch, and pushes the tag to the
configured remote.

Environment:
  REMOTE   Remote to push to (default: origin)
  BRANCH   Branch to fast-forward before tagging (default: current branch)
EOF
}

if [[ $# -lt 1 ]]; then
  usage >&2
  exit 1
fi

VERSION="$1"
shift || true
MESSAGE="${1:-Release ${VERSION}}"

TAG="$VERSION"
if [[ "$TAG" != v* ]]; then
  TAG="v$TAG"
fi

REMOTE="${REMOTE:-origin}"
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
BRANCH="${BRANCH:-$CURRENT_BRANCH}"

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || true)
if [[ -z "$REPO_ROOT" ]]; then
  echo "This script must be run inside a Git repository" >&2
  exit 1
fi
cd "$REPO_ROOT"

if [[ "$CURRENT_BRANCH" != "$BRANCH" ]]; then
  echo "Switching to branch $BRANCH"
  git checkout "$BRANCH"
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree or index has changes. Commit or stash before tagging." >&2
  exit 1
fi

if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Tag $TAG already exists locally. Delete it first if you wish to recreate it." >&2
  exit 1
fi

echo "Fetching latest state from $REMOTE"
git fetch "$REMOTE" --tags

echo "Fast-forwarding $BRANCH from $REMOTE"
git pull --ff-only "$REMOTE" "$BRANCH"

echo "Creating annotated tag $TAG"
git tag -a "$TAG" -m "$MESSAGE"

echo "Pushing tag $TAG to $REMOTE"
git push "$REMOTE" "$TAG"

echo "Tag $TAG pushed successfully."
