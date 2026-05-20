#!/usr/bin/env bash
set -euo pipefail

ROOT="/Users/paolo/workspace/gastown"
ENV_FILE="$ROOT/.env"
THROTTLE_PLIST="$HOME/Library/LaunchAgents/com.gastown.throttle.plist"
BOOST_PLIST="$HOME/Library/LaunchAgents/com.gastown.boost.plist"
THROTTLE_LABEL="com.gastown.throttle"
BOOST_LABEL="com.gastown.boost"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  source "$ENV_FILE"
  set +a
fi

WORK_START="${GASTOWN_WORK_START:-09:00}"
OFFHOURS_START="${GASTOWN_OFFHOURS_START:-18:00}"
WORK_WEEKDAYS="${GASTOWN_WORK_WEEKDAYS:-1,2,3,4,5}"

hour_from_hhmm() { echo "${1%%:*}"; }
min_from_hhmm() { echo "${1##*:}"; }

emit_calendar() {
  local hhmm="$1"
  local hh mm
  hh="$(hour_from_hhmm "$hhmm")"
  mm="$(min_from_hhmm "$hhmm")"
  IFS=',' read -r -a days <<<"$WORK_WEEKDAYS"
  for d in "${days[@]}"; do
    cat <<EOF
        <dict>
          <key>Weekday</key><integer>${d}</integer>
          <key>Hour</key><integer>${hh}</integer>
          <key>Minute</key><integer>${mm}</integer>
        </dict>
EOF
  done
}

write_plist() {
  local label="$1"
  local script_path="$2"
  local when="$3"
  local out="$4"
  cat >"$out" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>${label}</string>
    <key>ProgramArguments</key>
    <array>
      <string>/bin/zsh</string>
      <string>-lc</string>
      <string>cd ${ROOT} &amp;&amp; ${script_path} &gt;&gt; /tmp/${label}.log 2&gt;&amp;1</string>
    </array>
    <key>RunAtLoad</key>
    <false/>
    <key>StartCalendarInterval</key>
    <array>
$(emit_calendar "$when")
    </array>
    <key>StandardOutPath</key>
    <string>/tmp/${label}.out.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/${label}.err.log</string>
  </dict>
</plist>
EOF
}

install_jobs() {
  mkdir -p "$HOME/Library/LaunchAgents"
  write_plist "$THROTTLE_LABEL" "./scripts/gastown-throttle.sh" "$WORK_START" "$THROTTLE_PLIST"
  write_plist "$BOOST_LABEL" "./scripts/gastown-boost.sh" "$OFFHOURS_START" "$BOOST_PLIST"
  launchctl bootout "gui/$(id -u)/$THROTTLE_LABEL" >/dev/null 2>&1 || true
  launchctl bootout "gui/$(id -u)/$BOOST_LABEL" >/dev/null 2>&1 || true
  launchctl bootstrap "gui/$(id -u)" "$THROTTLE_PLIST"
  launchctl bootstrap "gui/$(id -u)" "$BOOST_PLIST"
  echo "installed: $THROTTLE_LABEL at $WORK_START, $BOOST_LABEL at $OFFHOURS_START (weekdays $WORK_WEEKDAYS)"
}

uninstall_jobs() {
  launchctl bootout "gui/$(id -u)/$THROTTLE_LABEL" >/dev/null 2>&1 || true
  launchctl bootout "gui/$(id -u)/$BOOST_LABEL" >/dev/null 2>&1 || true
  rm -f "$THROTTLE_PLIST" "$BOOST_PLIST"
  echo "removed: $THROTTLE_LABEL, $BOOST_LABEL"
}

status_jobs() {
  launchctl print "gui/$(id -u)/$THROTTLE_LABEL" 2>/dev/null || echo "$THROTTLE_LABEL not loaded"
  launchctl print "gui/$(id -u)/$BOOST_LABEL" 2>/dev/null || echo "$BOOST_LABEL not loaded"
}

case "${1:-}" in
  install) install_jobs ;;
  uninstall) uninstall_jobs ;;
  status) status_jobs ;;
  run-work-now) "$ROOT/scripts/gastown-throttle.sh" ;;
  run-offhours-now) "$ROOT/scripts/gastown-boost.sh" ;;
  *)
    echo "usage: $0 {install|uninstall|status|run-work-now|run-offhours-now}"
    exit 1
    ;;
esac
