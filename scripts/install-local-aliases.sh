#!/usr/bin/env bash
set -euo pipefail

ALIAS_FILE="/Users/paolo/workspace/gastown/scripts/gastown-aliases.zsh"
TARGET="${1:-$HOME/.zshrc}"
LINE="source ${ALIAS_FILE}"

touch "$TARGET"

if rg -Fq "$LINE" "$TARGET"; then
  echo "already installed in $TARGET"
  exit 0
fi

printf "\n# Gastown local helpers\n%s\n" "$LINE" >>"$TARGET"
echo "installed aliases in $TARGET"
