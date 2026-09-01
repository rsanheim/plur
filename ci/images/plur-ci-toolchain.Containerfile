# crow-image: localhost/plur-ci-toolchain:current
# crow-context: .

FROM docker.io/library/debian:trixie-slim

ARG USERNAME=plur
ARG USER_UID=1000
ARG USER_GID=${USER_UID}

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    MISE_DATA_DIR=/opt/mise \
    MISE_CONFIG_DIR=/opt/mise/config \
    MISE_CACHE_DIR=/opt/mise/cache \
    MISE_YES=1 \
    PATH=/opt/mise/shims:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

USER root

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      build-essential \
      ca-certificates \
      curl \
      git \
      jq \
      libffi-dev \
      libreadline-dev \
      libsqlite3-dev \
      libssl-dev \
      libyaml-dev \
      passwd \
      pkg-config \
      procps \
      ripgrep \
      sqlite3 \
      sudo \
      tmux \
      tzdata \
      unzip \
      util-linux \
      xz-utils \
      zlib1g-dev; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    groupadd --gid "${USER_GID}" "${USERNAME}"; \
    useradd --uid "${USER_UID}" --gid "${USER_GID}" --create-home --shell /bin/bash "${USERNAME}"; \
    echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > "/etc/sudoers.d/${USERNAME}"; \
    chmod 0440 "/etc/sudoers.d/${USERNAME}"

RUN set -eux; \
    curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh; \
    install -d -o "${USERNAME}" -g "${USERNAME}" \
      "${MISE_DATA_DIR}" \
      "${MISE_CONFIG_DIR}" \
      "${MISE_CACHE_DIR}"

RUN printf 'export PATH=/opt/mise/shims:$PATH\n' > /etc/profile.d/mise.sh

COPY --chown=${USERNAME}:${USERNAME} .mise.toml ${MISE_CONFIG_DIR}/config.toml
USER ${USERNAME}
RUN set -eux; \
    export HOME="/home/${USERNAME}"; \
    mise trust "${MISE_CONFIG_DIR}/config.toml"; \
    mise install --yes; \
    mise ls; \
    mise reshim; \
    mise cache clear

USER root

LABEL org.opencontainers.image.title="Plur Crow CI" \
      org.opencontainers.image.description="Toolchain image for Plur Crow workflows" \
      org.opencontainers.image.source="https://github.com/rsanheim/plur" \
      org.opencontainers.image.documentation="https://github.com/rsanheim/plur/tree/main/.crow"
