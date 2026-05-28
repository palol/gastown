#!/bin/sh
set -e

# Re-apply git/dolt config on every start so env var changes take effect
# even when the home volume already exists from a previous run.
if [ -n "$GIT_USER" ] && [ -n "$GIT_EMAIL" ]; then
    git config --global user.name "$GIT_USER"
    git config --global user.email "$GIT_EMAIL"
    git config --global credential.helper store
    dolt config --global --add user.name "$GIT_USER"
    dolt config --global --add user.email "$GIT_EMAIL"
fi

# Sync host gcloud credentials (mounted read-only at /mnt/gcloud-host) into the
# agent home. A direct mount onto ~/.config/gcloud can't work because gcloud/bq
# must write their own state (credentials.db, logs, active_config); copying gives
# them a writable, agent-owned copy that still reflects the host's auth + project.
if [ -d /mnt/gcloud-host ]; then
    mkdir -p /home/agent/.config
    rm -rf /home/agent/.config/gcloud
    if cp -a /mnt/gcloud-host /home/agent/.config/gcloud 2>/dev/null; then
        echo "Synced gcloud credentials from host."
    else
        echo "Warning: could not copy gcloud credentials from /mnt/gcloud-host"
    fi
fi

if [ ! -f /gt/mayor/town.json ]; then
    echo "Initializing Gas Town workspace at /gt..."
    /app/gastown/gt install /gt --git
else
    echo "Existing Gas Town workspace detected at /gt; skipping reinstall."
fi

(cd /gt && gt dolt start) || echo "Warning: could not start dolt server"

exec "$@"
