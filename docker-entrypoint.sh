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

# Lab SSH access: copy the dedicated key (mounted read-only at /mnt/ssh-host/gt_lab)
# into ~/.ssh with strict perms and write a scoped config. A read-only mount keeps
# host uid/perms, and SSH refuses keys that are group/world-readable — so we copy +
# chmod rather than use the mount in place. Only the hosts listed here are reachable;
# no other key material (broad id_rsa, bigfoot's own key) enters the container.
if [ -f /mnt/ssh-host/gt_lab ]; then
    mkdir -p /home/agent/.ssh && chmod 700 /home/agent/.ssh
    cp /mnt/ssh-host/gt_lab /home/agent/.ssh/gt_lab && chmod 600 /home/agent/.ssh/gt_lab
    cat > /home/agent/.ssh/config <<'EOF'
Host lva
    HostName 192.168.50.224
    User lookdeep
    IdentityFile /home/agent/.ssh/gt_lab
    IdentitiesOnly yes
Host lvupa0
    HostName 192.168.50.249
    User lookdeep
    IdentityFile /home/agent/.ssh/gt_lab
    IdentitiesOnly yes
Host lvupa1
    HostName 10.0.0.44
    User lookdeep
    IdentityFile /home/agent/.ssh/gt_lab
    IdentitiesOnly yes
Host bigfoot
    HostName 10.0.0.23
    User paolo
    IdentityFile /home/agent/.ssh/gt_lab
    IdentitiesOnly yes
Host *
    StrictHostKeyChecking accept-new
EOF
    chmod 600 /home/agent/.ssh/config
    echo "Configured scoped lab SSH access (lva, lvupa0, lvupa1, bigfoot)."
fi

# Claude Code auth: CLAUDE_CODE_OAUTH_TOKEN (long-lived setup-token) is the source
# of truth. A file-based ~/.claude/.credentials.json (e.g. from an accidental
# in-container `claude /login`) shadows the env token with a short-lived 8h OAuth
# token that can't refresh headless — this caused town-wide 401 outages
# (Jul 1-2 and Jul 9, 2026). Purge it on every start so the env token always wins.
if [ -f /home/agent/.claude/.credentials.json ]; then
    rm -f /home/agent/.claude/.credentials.json
    echo "Removed stale file-based Claude credentials; using CLAUDE_CODE_OAUTH_TOKEN."
fi

if [ ! -f /gt/mayor/town.json ]; then
    echo "Initializing Gas Town workspace at /gt..."
    /app/gastown/gt install /gt --git
else
    echo "Existing Gas Town workspace detected at /gt; skipping reinstall."
fi

(cd /gt && gt dolt start) || echo "Warning: could not start dolt server"

exec "$@"
