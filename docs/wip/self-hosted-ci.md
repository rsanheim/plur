# Self-Hosted CI for Plur

Research and setup notes for running CircleCI self-hosted runners with proper isolation.

## The Problem: Machine Runner Security

CircleCI's Machine Runner 3.0 runs jobs **directly on the host machine** with no isolation:

* Jobs execute as your user with full filesystem access
* Any code in a CI job (including from PRs) can access SSH keys, credentials, and local files
* No VM or container isolation is provided
* CircleCI disables self-hosted runners for public repos with "build forked PRs" enabled for this reason

### When Machine Runner is Acceptable

* Private repos where you control all code
* Trusted teams with no external contributors
* Testing/experimentation (like our initial setup)

### When You Need Isolation

* Public repos accepting external PRs
* Any scenario where untrusted code might run
* Production CI infrastructure

## Solution: Tart VM Isolation

[Tart](https://tart.run) is a virtualization tool for Apple Silicon that provides:

* Native performance via Apple's Virtualization.Framework
* OCI-compatible image registry (manage VMs like Docker images)
* Ephemeral VMs that can be cloned/destroyed per job
* SSH access for running commands inside VMs

### Security Model: True Hypervisor Isolation

**Tart VMs are real virtual machines, not Docker-style containers.**

Apple's Virtualization.framework is a Type 2 hypervisor providing hardware-level isolation:

* **Separate kernel**: Each VM runs its own OS kernel (no kernel sharing like Docker)
* **Memory isolation**: VMs cannot access each other's memory or host memory
* **Filesystem isolation**: VM has no direct access to host filesystem
* **Escape difficulty**: Breaking out requires exploiting the hypervisor itself (rare, serious vulnerability class)

**Docker/Linux containers by contrast:**
* Share the host kernel
* Isolation via namespaces, cgroups, seccomp
* Container escapes are a known attack class

Apple recently validated this model with their [Containerization framework](https://edera.dev/stories/apple-just-validated-hypervisor-isolated-containers-heres-what-that-means) (WWDC 2025) - they run one lightweight VM per container specifically because "no shared kernel, no namespace-based isolation, no where to escape to."

**Implication**: Running CircleCI machine runner directly inside a Tart VM (without another container layer) is secure enough. The hypervisor boundary is the security perimeter.

### Network Isolation Caveat

Default NAT networking doesn't fully isolate VMs from each other - [ARP spoofing between VMs is possible](https://medium.com/cirruslabs/isolating-network-between-tarts-macos-virtual-machines-9a4ae3dcf7be). Use **Softnet** mode for multi-tenant scenarios:

```bash
tart run --net-softnet vm-name
```

### Nested Virtualization Limitation

* **macOS guests**: Not supported (even on M3/M4 with hardware capability)
* **Linux guests**: Supported on M3+ chips with `tart run --nested`

This means you cannot run Docker-in-Tart for macOS VMs. See [discussion #701](https://github.com/cirruslabs/tart/discussions/701).

### System Requirements

* Apple Silicon Mac (M1/M2/M3/M4)
* macOS 13.0 (Ventura) or later
* ~25 GB disk space per VM image

### Installation

```bash
brew install cirruslabs/cli/tart
```

### Quick Start

```bash
# Clone a pre-built macOS image
tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest sequoia-base

# Run the VM (opens GUI window)
tart run sequoia-base

# SSH into the VM (default: admin/admin)
ssh admin@$(tart ip sequoia-base)
```

### Available Images

| Image | Contents |
|-------|----------|
| `macos-sequoia-vanilla` | Bare macOS installation |
| `macos-sequoia-base` | macOS + developer tools |
| `macos-sequoia-xcode` | macOS + Xcode IDE |

Also available: Sonoma, Ventura, Monterey variants, plus Linux (Ubuntu, Debian, Fedora)

### VM Configuration

```bash
# Set CPU/memory (default: 2 CPU, 4GB RAM)
tart set sequoia-base --cpu 4 --memory 8192

# Check VM IP for SSH
tart ip sequoia-base
```

### Headless Operation (Important for CI)

For macOS 15+ (Sequoia), Tart requires an unlocked keychain to run VMs. In headless/SSH sessions:

```bash
# Create and unlock keychain non-interactively
security create-keychain -p '' login.keychain
security unlock-keychain -p '' login.keychain
```

Or connect via Screen Sharing once to initialize the GUI session.

### Running Commands in VM

```bash
# Run a single command
ssh admin@$(tart ip sequoia-base) "uname -a"

# Or use sshpass for scripting
sshpass -p admin ssh -o "StrictHostKeyChecking no" admin@$(tart ip sequoia-base) "./run-tests.sh"
```

## Architecture: CircleCI Runner Inside Tart VM

The isolated setup would work like this:

```
┌─────────────────────────────────────────────┐
│ Mac Studio (Host)                           │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │ Tart VM (ephemeral)                   │  │
│  │                                       │  │
│  │  ┌─────────────────────────────────┐  │  │
│  │  │ CircleCI Runner                 │  │  │
│  │  │ * Receives jobs                 │  │  │
│  │  │ * Executes in isolated env      │  │  │
│  │  │ * No access to host filesystem  │  │  │
│  │  └─────────────────────────────────┘  │  │
│  │                                       │  │
│  │  mise, ruby, go, etc.                 │  │
│  └───────────────────────────────────────┘  │
│                                             │
└─────────────────────────────────────────────┘
```

For implementation details, see [self-hosted-ci-implementation.md](self-hosted-ci-implementation.md).

## Concurrency & Job Queuing

**Each machine runner agent can only execute one job at a time.** This is fundamental to how CircleCI machine runners work—it's not a configuration option.

### How It Works

1. The runner agent polls CircleCI for available tasks
2. When a task is available, the agent "claims" it
3. The agent executes the job
4. When complete, the agent polls for the next task

If multiple jobs target the same resource class (e.g., `rsanheim/mac-studio`) and only one runner agent is available, jobs queue and execute sequentially.

### Single-Task vs Continuous Mode

These modes control what happens *between* jobs, not concurrency:

* **Continuous mode** (default): Agent finishes a job, then polls for the next one
* **Single-task mode**: Agent finishes a job, then exits (useful for ephemeral/clean environments)

Both modes execute one job at a time per agent.

### Scaling Options

If job queuing becomes a bottleneck:

1. **Serialize the workflow**: Add job dependencies so jobs don't compete for the runner. Longer total time, but predictable.

2. **Run multiple agents in the VM**: Use Docker Compose with replicas, each container running its own runner agent. Requires managing resource contention within the VM.

3. **Multiple Tart VMs**: Run separate VMs, each with its own runner agent. More isolation, more resource overhead.

4. **Accept queuing**: For many workflows, sequential execution is fine. One runner handles jobs in order.

### References

* [CircleCI Runner FAQs](https://circleci.com/docs/runner-faqs/)
* [Single Task Runner - CircleCI Field Guide](https://fieldguide.circleci-fieldeng.com/runner/single-task/)
* [Scalable Machine Runner](https://github.com/CircleCI-Labs/scalable-machine-runner)

## Historical Reference: Host-Based Runner (Deprecated)

This section documents our initial approach of running the CircleCI runner directly on the host Mac Studio. **This approach is no longer used** because it lacks isolation—any CI job could access the host filesystem, SSH keys, and credentials.

We now run the runner inside a Tart VM. See [self-hosted-ci-implementation.md](self-hosted-ci-implementation.md) for the current setup.

The information below is preserved for reference in case you need to understand the old approach or troubleshoot legacy configurations.

### Installation

```bash
brew tap circleci-public/circleci
brew install circleci-runner
```

### Paths

* **Logs**: `~/Library/Logs/com.circleci.runner/`
* **Config**: `~/Library/Preferences/com.circleci.runner/config.yaml`
* **LaunchAgent**: `~/Library/LaunchAgents/com.circleci.runner.plist`
* **Wrapper script**: `~/.local/bin/circleci-runner-wrapper`

### Configuration

Config at `~/Library/Preferences/com.circleci.runner/config.yaml`:

```yaml
runner:
  name: mac-studio
  working_directory: "/Users/rsanheim/Library/com.circleci.runner/workdir"
  cleanup_working_directory: true
# auth_token provided via CIRCLECI_RUNNER_API_AUTH_TOKEN env var
```

### Token Injection via 1Password

The wrapper script (`~/.local/bin/circleci-runner-wrapper`) fetches the token from 1Password:

```bash
#!/bin/bash
export CIRCLECI_RUNNER_API_AUTH_TOKEN="$(/opt/homebrew/bin/op read 'op://Private/circle ci self hosted runner/credential')"
exec /opt/homebrew/bin/circleci-runner machine --config /Users/rsanheim/Library/Preferences/com.circleci.runner/config.yaml
```

### macOS Security

```bash
# Check notarization
spctl -a -vvv -t install "$(brew --prefix)/bin/circleci-runner"

# Accept notarization headlessly
sudo xattr -r -d com.apple.quarantine "$(brew --prefix)/bin/circleci-runner"
```

### Service Management

```bash
# Load/reload the LaunchAgent
PLIST=~/Library/LaunchAgents/com.circleci.runner.plist
launchctl load $PLIST || (launchctl unload $PLIST && launchctl load $PLIST)

# Check status
launchctl list | grep circleci

# View logs
tail -f ~/Library/Logs/com.circleci.runner/runner.log
```

### CircleCI Workflow

The `selfhosted-test` workflow in `.circleci/config.yml` runs on resource class `rsanheim/mac-studio`

## References

* [Tart Quick Start](https://tart.run/quick-start/)
* [Tart FAQ](https://tart.run/faq/)
* [Tart GitHub](https://github.com/cirruslabs/tart)
* [CircleCI Runner Overview](https://circleci.com/docs/runner-overview/)
* [CircleCI Runner FAQs](https://circleci.com/docs/runner-faqs)
* [CircleCI Config Policies for Runners](https://circleci.com/docs/config-policies-for-self-hosted-runner)
