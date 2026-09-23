#!/bin/sh
# PostToolUse(Edit|Write): format the file that was just written.
#   *.go                    -> gofmt -w        (skipped when gofmt is absent)
#   *.ts|tsx|css|json|md    -> the nearest node_modules/.bin/prettier --write
#
# Contract: NEVER fail and NEVER print. A missing tool, an unparsable payload or a path
# outside a repo must all exit 0 silently, so editing is never blocked.
#
# Identical copies live in CinemaProject/.claude/hooks/, BackEnd-CP/.claude/hooks/ and
# FrontEnd-CP/.claude/hooks/ because hooks are NOT inherited from a parent directory's
# settings.json. Keep the three in sync.
set -u

payload=$(cat 2>/dev/null) || exit 0
[ -n "$payload" ] || exit 0

command -v python3 >/dev/null 2>&1 || exit 0

# The stdin field holding the edited path is not pinned by the docs, so accept every shape.
file=$(printf '%s' "$payload" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
except Exception:
    sys.exit(0)
if not isinstance(d, dict):
    sys.exit(0)
for s in (d.get("tool_input"), d.get("toolInput"), d.get("tool_response"), d.get("toolResponse"), d):
    if isinstance(s, list):
        s = s[0] if s and isinstance(s[0], dict) else None
    if not isinstance(s, dict):
        continue
    for k in ("file_path", "filePath", "path", "notebook_path", "notebookPath"):
        v = s.get(k)
        if isinstance(v, str) and v.strip():
            print(v.strip())
            sys.exit(0)
' 2>/dev/null) || exit 0

[ -n "$file" ] || exit 0
[ -f "$file" ] || exit 0

# Never reformat harness configuration: prettier would rewrite settings.json and
# the YAML frontmatter of rules/skills on every edit, which is surprising noise.
case "$file" in
  */.claude/*) exit 0 ;;
esac

case "$file" in
  *.go)
    command -v gofmt >/dev/null 2>&1 || exit 0
    gofmt -w "$file" >/dev/null 2>&1
    exit 0
    ;;
  *.ts|*.tsx|*.css|*.json|*.md)
    dir=$(dirname "$file")
    # Walk up looking for a repo-local prettier. Never install anything.
    while [ -n "$dir" ] && [ "$dir" != "/" ]; do
      if [ -x "$dir/node_modules/.bin/prettier" ]; then
        "$dir/node_modules/.bin/prettier" --write "$file" >/dev/null 2>&1
        exit 0
      fi
      parent=$(dirname "$dir")
      [ "$parent" = "$dir" ] && break
      dir=$parent
    done
    exit 0
    ;;
esac

exit 0
