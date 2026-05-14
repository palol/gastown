#!/usr/bin/env bash
set -euo pipefail

REQUIRED_ABS='/Users/paolo/brain/Work/_META/coding-agent-contract.md'
REQUIRED_REL='_META/coding-agent-contract.md'

FILES=(
  "/Users/paolo/workspace/gastown/AGENTS.md"
  "/Users/paolo/workspace/beads/AGENTS.md"
  "/Users/paolo/workspace/beads/cmd/bd/AGENTS.md"
  "/Users/paolo/workspace/beads/plugins/beads/skills/beads/resources/AGENTS.md"
  "/Users/paolo/workspace/ld-agent-lab/AGENTS.md"
  "/Users/paolo/brain/Work/GEMINI.md"
  "/Users/paolo/gt/CLAUDE.md"
  "/Users/paolo/gt/lightfoot/AGENTS.md"
  "/Users/paolo/gt/lightfoot_ml_tools/AGENTS.md"
  "/Users/paolo/gt/ld_agent_lab/AGENTS.md"
  "/Users/paolo/gt/ld_fiftyone/AGENTS.md"
)

fail=0

for file in "${FILES[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "MISSING: $file"
    fail=1
    continue
  fi

  if ! rg -Fq "$REQUIRED_ABS" "$file" && ! rg -Fq "$REQUIRED_REL" "$file"; then
    echo "MISSING CONTRACT REF: $file"
    fail=1
  else
    echo "OK: $file"
  fi
done

exit "$fail"
