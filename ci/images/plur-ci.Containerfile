# Plur's CI toolchain image. .mise.toml is the single source of truth for every
# pinned tool, so a toolchain bump is a one-line change there and a rebuild.
#
# This file is the definition only; building and publishing it is handled by
# shared CI tooling (rsanheim/infra#96). Build it by hand with:
#   podman build -t plur-ci -f ci/images/plur-ci.Containerfile .
#
# The build context is the repository root, because the image bakes .mise.toml.
#
# crow-image: localhost/plur-ci:current
# crow-context: .
FROM debian:trixie-slim

ARG USERNAME=plur
ARG USER_UID=1000
ARG USER_GID=${USER_UID}
# Keep in sync with BUNDLED WITH in Gemfile.lock (root and fixtures) so bundle
# invocations skip bundler's self-reinstall-and-restart on version drift.
ARG BUNDLER_VERSION=4.0.17

ENV DEBIAN_FRONTEND=noninteractive \
    MISE_DATA_DIR=/opt/mise \
    MISE_CONFIG_DIR=/opt/mise/config \
    MISE_CACHE_DIR=/opt/mise/cache \
    MISE_YES=1 \
    PATH=/opt/mise/shims:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

USER root

# System packages for the build and the suite: build-essential + pkg-config +
# libyaml-dev + libsqlite3-dev compile native gems (psych, sqlite3); the
# sqlite3 CLI verifies rails-fixture migrations; tmux is the terminal the
# watch REPL specs drive plur through; passwd + util-linux provide
# useradd/runuser for bench.yaml's unprivileged bench user; xz-utils unpacks
# mise's tarballs.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      build-essential \
      ca-certificates \
      curl \
      git \
      jq \
      libsqlite3-dev \
      libyaml-dev \
      passwd \
      pkg-config \
      sqlite3 \
      tmux \
      util-linux \
      xz-utils; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    groupadd --gid "${USER_GID}" "${USERNAME}"; \
    useradd --uid "${USER_UID}" --gid "${USER_GID}" --create-home --shell /bin/bash "${USERNAME}"

RUN set -eux; \
    curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh; \
    install -d -o "${USERNAME}" -g "${USERNAME}" \
      "${MISE_DATA_DIR}" \
      "${MISE_CONFIG_DIR}" \
      "${MISE_CACHE_DIR}"

# Login shells run /etc/profile, which resets PATH and drops the shims.
RUN printf 'export PATH=/opt/mise/shims:$PATH\n' > /etc/profile.d/mise.sh

# Ruby linting requires a UTF-8 default; C.UTF-8 needs no locales package.
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

# Pre-bake the mise toolchain from .mise.toml (ruby, go, goreleaser,
# shellcheck, hyperfine, govulncheck, python). `mise install` is the gate -
# non-zero exit if any pinned tool fails; `mise ls` records the resolved set.
COPY --chown=${USERNAME}:${USERNAME} .mise.toml ${MISE_CONFIG_DIR}/config.toml
USER ${USERNAME}
RUN set -eux; \
    export HOME="/home/${USERNAME}"; \
    mise trust "${MISE_CONFIG_DIR}/config.toml"; \
    mise install; \
    mise ls; \
    mise reshim; \
    mise cache clear; \
    gem install bundler -v "${BUNDLER_VERSION}"
USER root
