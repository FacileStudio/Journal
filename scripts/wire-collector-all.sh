#!/usr/bin/env bash
# Wire Journal collector to all sibling Facile projects that have a docker-compose.yml
# Usage: run from the root of the Facile monorepo (where this script lives)
# It copies the prepared docker-compose.collector.yml into each project that lacks it.

set -euo pipefail

# Path to the collector snippet (relative to the script location)
SNIPPET="$(dirname "${BASH_SOURCE[0]}")/../docker-compose.collector.yml"

if [[ ! -f "$SNIPPET" ]]; then
  echo "Collector snippet not found at $SNIPPET" >&2
  exit 1
fi

# Find sibling directories containing a docker-compose.yml (any name)
for d in */ ; do
  # Skip this repo (Journal) – it already has the file
  [[ "$d" == "./" ]] && continue
  if [[ -f "${d}docker-compose.yml" ]]; then
    if [[ -f "${d}docker-compose.collector.yml" ]]; then
      echo "${d} already has collector config – skipping"
    else
      echo "Copying collector config into ${d}"
      cp "$SNIPPET" "${d}docker-compose.collector.yml"
    fi
  fi
done

echo "Done."
