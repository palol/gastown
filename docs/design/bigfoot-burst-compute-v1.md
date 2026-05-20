# Laptop-Canonical GT, Bigfoot Burst Compute (v1)

## Decision

Canonical Gas Town control plane stays on laptop.
Bigfoot is burst compute executor only.

## Scope

In scope:
- Compute routing primitive: `compute:bigfoot` label and `--compute-target auto|local|bigfoot`.
- Immutable dispatch audit fields on work bead:
  - `dispatch_run_id`
  - `routed_host`
  - `dispatch_started_at`
  - `resource_class`
- Mutagen Project config (`mutagen.yml`) with safe one-way flows.

Out of scope in this patch:
- Full remote run daemon.
- Remote/local reconciliation state machine execution.
- Remote write sandbox daemon enforcement.

## Routing Contract

- Default: `--compute-target=auto`
- `auto` behavior:
  - If bead has label `compute:bigfoot`, route to rig `bigfoot`.
  - Else use requested local rig.
- `--compute-target=local` forces local.
- `--compute-target=bigfoot` forces bigfoot.

Routing applies in direct sling path and scheduler/queue dispatch path.

## Audit Contract

On dispatch, GT writes these bead fields:

- `compute_target`: resolved target (`local` or `bigfoot`)
- `dispatch_run_id`: run identifier (from `GT_RUN`, else UUID)
- `routed_host`: local hostname or bigfoot host
- `dispatch_started_at`: RFC3339 UTC timestamp
- `resource_class`: `local` or `burst`

Immutability rule:
- `dispatch_run_id`, `routed_host`, `dispatch_started_at`, `resource_class` write once.
- Later sling updates do not overwrite existing values.

## Mutagen Contract

Use Mutagen **Project** file (`mutagen.yml`), not mutagen-compose.

Defined sessions:
- Local source to bigfoot workspace (`one-way-safe`)
- Bigfoot runs inbox back to laptop (`one-way-safe`)

Never sync:
- `.dolt*`
- `.beads`
- secrets/env files
- dataset/checkpoint/blob trees

Only collect from bigfoot runs inbox:
- `manifest.json`
- `summary.md`
- `patch.diff`

## Run State Machine (target v1 contract)

Planned remote run lifecycle:

`queued -> preflight -> syncing -> dispatched -> running -> collecting -> succeeded|failed|cancelled|abandoned`

This patch documents contract and implements routing + audit substrate needed before full remote runner.

## Current Guardrails Implemented

- Remote control-plane guard:
  - `gt sling` blocked when `GT_REMOTE_EXECUTOR=1` or `GT_CONTROL_PLANE_MODE=remote-executor`
  - `gt mail send` blocked in same mode
  - `gt close` blocked in same mode
- Scheduler sling context run-state transitions now tracked:
  - `queued -> preflight -> syncing -> dispatched -> running -> collecting -> succeeded`
  - failure path sets `failed`
  - stale-heartbeat / collect-timeout paths set `abandoned`
- Artifact collect validation + hash audit for bigfoot collect phase:
  - requires `manifest.json` and `summary.md`
  - optional `patch.diff`
  - stores `manifest_sha256`, `summary_sha256`, `patch_sha256`
  - closes context with `collected` or `collect-validation-failed`

## Failure Handling Targets

- Missing bigfoot rig: sling fails fast via existing rig resolution.
- Dispatch metadata write failures: warn, do not abort dispatch.
- Batch dispatch wake-up: wake witness for each routed rig used in batch.

## Test Checklist

- Route test: bead with `compute:bigfoot` routes to bigfoot in auto mode.
- Override test: `--compute-target=local` ignores label.
- Audit test: dispatch fields written; immutable fields do not overwrite.
- Scheduler reconstruction test: context preserves `compute_target`.
