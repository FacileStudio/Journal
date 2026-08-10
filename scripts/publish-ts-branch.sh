#!/usr/bin/env sh
#
# Publish sdk/browser to the `ts` branch, which is what
# `bun add github:FacileStudio/Journal#ts` installs.
#
#   sh scripts/publish-ts-branch.sh            # split and push
#   sh scripts/publish-ts-branch.sh --dry-run  # split locally, push nothing
#
# A github: dependency installs the repository *root*, so Bun needs to find a
# package.json there. The `ts` branch therefore holds the contents of
# sdk/browser at its root, produced by git subtree split — never by hand.
#
# dist/ is committed for the same reason: nothing runs tsc at install time, so
# a consumer gets exactly the build that was pushed. This script rebuilds and
# refuses to publish a dist/ that does not match src/.
#
# Consequence worth stating once: #ts is a moving branch reference, not a
# version. A push here is a release to every consumer's next install.

set -eu

PREFIX="sdk/browser"
BRANCH="ts"
REMOTE="${REMOTE:-origin}"

dry_run=0
case "${1:-}" in
--dry-run) dry_run=1 ;;
"") ;;
*)
  echo "usage: $0 [--dry-run]" >&2
  exit 2
  ;;
esac

cd "$(git rev-parse --show-toplevel)"

if [ -n "$(git status --porcelain -- "$PREFIX")" ]; then
  echo "publish: $PREFIX has uncommitted changes; commit them first" >&2
  exit 1
fi

echo "==> rebuilding $PREFIX"
(
  cd "$PREFIX"
  [ -d node_modules ] || bun install >/dev/null
  bun run typecheck
  bun test
  bun run build >/dev/null
)

if [ -n "$(git status --porcelain -- "$PREFIX/dist")" ]; then
  echo "publish: $PREFIX/dist is out of date with src; commit the rebuilt dist first" >&2
  exit 1
fi

echo "==> splitting $PREFIX onto $BRANCH"
git branch -D "$BRANCH" 2>/dev/null || true
git subtree split --prefix="$PREFIX" -b "$BRANCH"

if [ "$dry_run" -eq 1 ]; then
  echo "publish: dry run, leaving $BRANCH local"
  exit 0
fi

echo "==> pushing $BRANCH to $REMOTE"
git push --force "$REMOTE" "$BRANCH:$BRANCH"
echo "published: bun add github:FacileStudio/Journal#$BRANCH"
