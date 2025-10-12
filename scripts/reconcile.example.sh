#!/usr/bin/env bash
set -euo pipefail

repoRoot="$1"
target="$2"      # file or directory
policy="$3"

RUNTIME="${MD_RUNTIME:-docker}"
case "$RUNTIME" in
  docker) CMD="docker compose" ;;
  podman) CMD="podman compose" ;;
  *) echo "unknown runtime: $RUNTIME" >&2; exit 2 ;;
esac

# Resolve target to a compose file + working dir
if [ -d "$target" ]; then
  cd "$target"
  file="$(ls -1 \
    docker-compose.yml docker-compose.yaml \
    compose.yml compose.yaml \
    *compose*.yml 2>/dev/null | head -n1 || true)"
  if [ -z "${file:-}" ]; then
    echo "no compose file found in $target" >&2
    exit 2
  fi
else
  cd "$(dirname "$target")"
  file="$(basename "$target")"
fi

# Do the thing
$CMD -f "$file" pull
$CMD -f "$file" up -d
echo "[reconcile] applied $PWD/$file (policy=$policy)"
