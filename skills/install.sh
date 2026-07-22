#!/usr/bin/env bash
# Symlink the things-cli skill into local agent skill directories.
set -euo pipefail

SKILL_NAME="things-cli"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$SCRIPT_DIR/$SKILL_NAME"

if [[ ! -d "$SRC" ]]; then
  echo "error: skill source not found at $SRC" >&2
  exit 1
fi

SCOPE="${SKILL_SCOPE:-home}"
if [[ "$SCOPE" == "project" ]]; then
  BASE="$(cd "$SCRIPT_DIR/.." && pwd)"
else
  BASE="$HOME"
fi

TARGET_DIRS=(".claude/skills" ".agents/skills")
UNINSTALL=0
[[ "${1:-}" == "--uninstall" ]] && UNINSTALL=1

for rel in "${TARGET_DIRS[@]}"; do
  dest_dir="$BASE/$rel"
  dest="$dest_dir/$SKILL_NAME"
  if [[ "$UNINSTALL" == "1" ]]; then
    if [[ -L "$dest" ]]; then
      rm "$dest"
      echo "removed $dest"
    fi
    continue
  fi
  mkdir -p "$dest_dir"
  if [[ -L "$dest" ]]; then
    rm "$dest"
  elif [[ -e "$dest" ]]; then
    echo "error: $dest exists and is not a symlink; leaving it alone" >&2
    continue
  fi
  ln -s "$SRC" "$dest"
  echo "linked $dest -> $SRC"
done
