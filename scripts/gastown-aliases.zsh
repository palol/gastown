# Gastown local helper aliases/functions.
# Usage:
#   source /Users/paolo/workspace/gastown/scripts/gastown-aliases.zsh

_gt_root="/Users/paolo/workspace/gastown"
_gt_env="$_gt_root/.env"
_gt_compose=(docker compose --env-file "$_gt_env" -f "$_gt_root/docker-compose.yml" -f "$_gt_root/docker-compose.local.yml" -f "$_gt_root/docker-compose.auth-host.yml")

gtm() { "${_gt_compose[@]}" exec gastown gt mayor attach; }
function bd() { "${_gt_compose[@]}" exec -T gastown bd "$@"; }
gtd() {
  local port
  port="$(awk -F= '/^DASHBOARD_PORT=/{print $2}' "$_gt_env" 2>/dev/null | tr -d '"' | tail -n1)"
  [[ -z "$port" ]] && port=8080
  if ! curl -sf --connect-timeout 1 "http://localhost:${port}" >/dev/null 2>&1; then
    echo "Starting dashboard on port ${port}..."
    "${_gt_compose[@]}" exec -d gastown gt dashboard --bind 0.0.0.0 --port "$port"
    local i=0
    until curl -sf --connect-timeout 1 "http://localhost:${port}" >/dev/null 2>&1; do
      sleep 0.5
      (( i++ ))
      [[ $i -ge 20 ]] && { echo "Dashboard did not start"; return 1; }
    done
  fi
  open "http://localhost:${port}"
}
gtf() { "${_gt_compose[@]}" exec gastown gt feed; }

gstart() { "$_gt_root/scripts/gastown-boost.sh"; }
gpause() { "$_gt_root/scripts/gastown-throttle.sh"; }
gstop() { "${_gt_compose[@]}" stop gastown; }
grebuild() { "${_gt_compose[@]}" build gastown && "${_gt_compose[@]}" up -d --no-build gastown; }
gup() { "${_gt_compose[@]}" up -d --no-build gastown; }
gstatus() { "${_gt_compose[@]}" exec -T gastown gt scheduler status; }

gsched-on() { "$_gt_root/scripts/gastown-schedule-launchd.sh" install; }
gsched-off() { "$_gt_root/scripts/gastown-schedule-launchd.sh" uninstall; }
gsched-status() { "$_gt_root/scripts/gastown-schedule-launchd.sh" status; }
gsched-work-now() { "$_gt_root/scripts/gastown-schedule-launchd.sh" run-work-now; }
gsched-offhours-now() { "$_gt_root/scripts/gastown-schedule-launchd.sh" run-offhours-now; }

# Auth refresh happens on host because auth mounts are read-only in container.
gauth-gh() { gh auth login; }
gauth-claude() { claude auth login; }
gauth-gemini() { gemini auth login; }
gauth-cursor() { cursor-agent login; }

ghelp() {
  cat <<'EOF'
Gastown aliases
  gt <cmd>            run gt in container
  gts                 gt status
  gtv                 gt vitals
  gtm                 gt mayor attach (interactive)
  gtf                 gt feed
  gtd                 open dashboard in browser

Lifecycle
  gup                 start container (no rebuild)
  grebuild            rebuild image + restart container
  gstart              boost (gastown-boost.sh)
  gpause              throttle (gastown-throttle.sh)
  gstop               stop container
  gstatus             scheduler status

Scheduler (launchd)
  gsched-on           install schedule
  gsched-off          uninstall schedule
  gsched-status       schedule status
  gsched-work-now     run work cycle now
  gsched-offhours-now run offhours cycle now

Build
  gtbuild             make safe-install
  gtinstall           make install

Beads
  bd <cmd>            run bd in container
  bd stats            beads stats
  bd list             list issues
  bd vc commit        commit beads changes

Auth (host-side)
  gauth-gh            gh auth login
  gauth-claude        claude auth login
  gauth-gemini        gemini auth login
  gauth-cursor        cursor-agent login
EOF
}
