# Run with
# docker build -t gastown:latest -f Dockerfile .
FROM docker/sandbox-templates:claude-code

ARG GO_VERSION=1.25.8

USER root

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    git-lfs \
    sqlite3 \
    tmux \
    curl \
    ripgrep \
    zsh \
    gh \
    netcat-openbsd \
    tini \
    vim \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* \
    && git lfs install --system

# Install Go from official tarball (apt golang-go is too old)
RUN ARCH=$(dpkg --print-architecture) && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" | tar -C /usr/local -xz
ENV PATH="/app/gastown:/usr/local/go/bin:/home/agent/go/bin:/home/agent/.local/bin:${PATH}"

# Install beads (bd) and dolt
RUN curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
RUN curl -fsSL https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash

# Set up directories
RUN mkdir -p /app /gt /gt/.dolt-data && chown -R agent:agent /app /gt

# Environment setup for bash and zsh
RUN echo 'export PATH="/app/gastown:$PATH"' >> /etc/profile.d/gastown.sh && \
    echo 'export PATH="/app/gastown:$PATH"' >> /etc/zsh/zshenv
RUN echo 'export COLORTERM="truecolor"' >> /etc/profile.d/colorterm.sh && \
    echo 'export COLORTERM="truecolor"' >> /etc/zsh/zshenv
RUN echo 'export TERM="xterm-256color"' >> /etc/profile.d/term.sh && \
    echo 'export TERM="xterm-256color"' >> /etc/zsh/zshenv

USER agent

# Install Gemini CLI and Cursor Agent CLI for auth/tool parity with host.
RUN npm install -g @google/gemini-cli
RUN curl -fsSL https://cursor.com/install | bash
USER root
RUN CURSOR_ENTRY="$(readlink -f /home/agent/.local/bin/agent)" && \
    CURSOR_DIR="$(dirname "$CURSOR_ENTRY")" && \
    mkdir -p /opt/cursor-cli && \
    cp -a "$CURSOR_DIR"/. /opt/cursor-cli/ && \
    ln -sf /opt/cursor-cli/cursor-agent /usr/local/bin/cursor-agent && \
    ln -sf /opt/cursor-cli/agent /usr/local/bin/agent
USER agent

COPY --chown=agent:agent . /app/gastown

RUN cd /app/gastown && make build

COPY --chown=agent:agent docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

WORKDIR /gt

ENTRYPOINT ["tini", "--", "/app/docker-entrypoint.sh"]
CMD ["sleep", "infinity"]
