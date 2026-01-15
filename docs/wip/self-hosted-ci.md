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

### Setup Steps (Future Work)

1. Create a base Tart VM with required tools (mise, ruby, go)
2. Install CircleCI runner inside the VM
3. Create a wrapper script on host that:
   * Clones a fresh VM from the base image
   * Starts the VM
   * Runner inside VM claims and executes jobs
   * VM is destroyed after job completes (ephemeral)

### Considerations

* **Persistence**: Need to decide if VMs are truly ephemeral (clone per job) or long-running
* **Image management**: Custom image with pre-installed tools vs. install at runtime
* **Startup time**: VM clone + boot adds overhead (~10-30 seconds)
* **Resource allocation**: CPU/memory limits per VM

## Current State (Machine Runner on Host)

We have a working (but unisolated) setup running directly on the Mac Studio:

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
