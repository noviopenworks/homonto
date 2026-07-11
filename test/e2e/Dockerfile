# Real dual-binary E2E image. Builds homonto AND onto from source, then installs
# the ACTUAL Claude Code and OpenCode CLIs via their official installers, so the
# suites exercise homonto's projection and onto's workflow against the real
# tools. Credentials are never baked in — they are mounted read-only at run time
# by scripts/e2e-matrix.sh. Built once, run per matrix cell via entrypoint.sh.
FROM golang:1.23-bookworm

# Runtime deps: curl for the installers, jq/python3 for the suite assertions,
# git for onto's dirty-worktree checks, unzip for the opencode installer.
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates git jq python3 unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Cache module downloads separately from source.
COPY go.mod go.sum ./
RUN go mod download

# Build both binaries from the full source tree.
COPY . .
RUN go build -o /usr/local/bin/homonto . \
    && go build -o /usr/local/bin/onto ./cmd/onto

# --- install the real tools (native installers, no npm needed) ---------------
ENV HOME=/root
# Claude Code native installer -> /root/.local/bin/claude
RUN curl -fsSL https://claude.ai/install.sh | bash
# OpenCode installer -> /root/.opencode/bin/opencode
RUN curl -fsSL https://opencode.ai/install | bash
ENV PATH="/root/.local/bin:/root/.opencode/bin:${PATH}"

# Verify all four binaries resolve at build time (fail fast on a broken image).
RUN homonto version && onto version \
    && claude --version && opencode --version

COPY test/e2e/entrypoint.sh /usr/local/bin/e2e-entrypoint
COPY test/e2e/suites /opt/e2e-suites
RUN chmod +x /usr/local/bin/e2e-entrypoint /opt/e2e-suites/*.sh

ENTRYPOINT ["/usr/local/bin/e2e-entrypoint"]
