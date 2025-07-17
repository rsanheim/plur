FROM ubuntu:24.04

# Prevent interactive prompts during apt-get
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    vim \
    less \
    strace \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

# Install hyperfine for benchmarking (handle both amd64 and arm64)
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
        wget https://github.com/sharkdp/hyperfine/releases/download/v1.18.0/hyperfine_1.18.0_amd64.deb && \
        dpkg -i hyperfine_1.18.0_amd64.deb && \
        rm hyperfine_1.18.0_amd64.deb; \
    elif [ "$ARCH" = "arm64" ]; then \
        wget https://github.com/sharkdp/hyperfine/releases/download/v1.18.0/hyperfine_1.18.0_arm64.deb && \
        dpkg -i hyperfine_1.18.0_arm64.deb && \
        rm hyperfine_1.18.0_arm64.deb; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi

# Install Go 1.24.4 (handle both amd64 and arm64)
RUN ARCH=$(dpkg --print-architecture) && \
    GO_ARCH=$([ "$ARCH" = "arm64" ] && echo "arm64" || echo "amd64") && \
    wget https://go.dev/dl/go1.24.4.linux-${GO_ARCH}.tar.gz && \
    tar -C /usr/local -xzf go1.24.4.linux-${GO_ARCH}.tar.gz && \
    rm go1.24.4.linux-${GO_ARCH}.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install Ruby 3.4
RUN apt-get update && apt-get install -y \
    autoconf \
    bison \
    patch \
    libssl-dev \
    libyaml-dev \
    libreadline6-dev \
    zlib1g-dev \
    libgmp-dev \
    libncurses5-dev \
    libffi-dev \
    libgdbm6 \
    libgdbm-dev \
    libdb-dev \
    uuid-dev \
    && rm -rf /var/lib/apt/lists/*

# Build Ruby 3.4 from source
RUN cd /tmp \
    && wget https://cache.ruby-lang.org/pub/ruby/3.4/ruby-3.4.4.tar.gz \
    && tar xzf ruby-3.4.4.tar.gz \
    && cd ruby-3.4.4 \
    && ./configure --prefix=/usr/local --enable-shared \
    && make -j$(nproc) \
    && make install \
    && cd / \
    && rm -rf /tmp/ruby-3.4.4*

# Install bundler
RUN gem install bundler

# Set up workspace
WORKDIR /workspace

# Create directories that will be excluded from volume mounts
RUN mkdir -p /workspace/references /workspace/vendor

# Set up non-root user (optional but good practice)
RUN useradd -m -s /bin/bash plur
RUN chown -R plur:plur /workspace

# Default to bash shell
CMD ["/bin/bash"]