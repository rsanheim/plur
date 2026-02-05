# Tart VM for Self-Hosted CI

This documents our self-hosted CI infrastructure using [Tart](https://tart.run/) VMs on Apple Silicon.

For the full implementation plan and status, see [self-hosted-ci-implementation.md](../wip/self-hosted-ci-implementation.md).

## Overview

Tart is a virtualization toolset for Apple Silicon that uses Apple's native Virtualization.framework. We run our CircleCI self-hosted runner *inside* a Tart VM to provide hypervisor-level isolation—CI jobs cannot access the host filesystem, SSH keys, or credentials.

**Why not run the runner directly on the host?** CircleCI's Machine Runner 3.0 has no built-in isolation. Any code in a CI job (including from PRs) would have full access to the host. See [self-hosted-ci.md](../wip/self-hosted-ci.md) for the security model details.

## Our VM: plur-runner

| Property | Value |
|----------|-------|
| VM Name | `plur-runner` |
| Base Image | macOS Sequoia (`ghcr.io/cirruslabs/macos-sequoia-base`) |
| Resources | 6 CPU, 16GB RAM, 60GB disk |
| User | `admin` |
| SSH Key | `~/.ssh/ci-vm-key` |

## Managing the VM

Use native `tart` commands for VM lifecycle management:

```bash
# List all VMs and their status
tart list

# Start VM headless (for CI/background work)
tart run --no-graphics plur-runner &

# Start VM with GUI window (useful for debugging)
tart run plur-runner

# Stop VM gracefully
tart stop plur-runner

# Get current VM IP address
tart ip plur-runner

# Clone a new VM from registry
tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest my-new-vm
```

### Resizing VMs

VMs must be stopped before resizing. CPU and RAM changes take effect immediately on next boot.

```bash
# Stop the VM first
tart stop plur-runner

# Change CPU and RAM
tart set plur-runner --cpu 6 --memory 16384

# Increase disk size (can only grow, not shrink)
tart set plur-runner --disk-size 60

# Start VM - disk expansion happens automatically (APFS)
tart run --no-graphics plur-runner &
```

### Installation

If you need to set up Tart on a new host:

```bash
brew install cirruslabs/cli/tart
```

## What's Running in the VM

### Development Tools (via mise)

The VM uses [mise](https://mise.jdx.dev/) for version management. Tools are defined in `.mise.toml`:

* Ruby 4
* Go 1.25
* Python 3

Run `script/ci-host-setup` inside the VM to install/update all tools.

### CI Infrastructure

* **CircleCI Runner**: Claims jobs from resource class `rsanheim/mac-studio`
* **Runner config**: `~/Library/Preferences/com.circleci.runner/config.yaml`
* **Auto-start**: LaunchAgent at `~/Library/LaunchAgents/com.circleci.runner.plist`

### Installed via Homebrew

* `circleci-runner` - CI job execution
* `goreleaser` - Building releases
* `bash` (modern 4+) - For script compatibility

## Debugging via SSH

Use the `tart-vm` command for SSH and SCP operations:

```bash
# Interactive shell
tart-vm ssh plur-runner

# Run a command
tart-vm ssh plur-runner "uname -a"

# With TTY (for interactive commands)
tart-vm ssh plur-runner -t "top"

# Copy files to VM (colon prefix = VM side)
tart-vm scp plur-runner local-file.txt :~/remote-path/

# Copy files from VM
tart-vm scp plur-runner :~/remote-file.txt ./local-path/

# Recursive copy
tart-vm scp plur-runner -r local-dir :~/
```

### Direct SSH

If `tart-vm` isn't available:

```bash
ssh -i ~/.ssh/ci-vm-key admin@$(tart ip plur-runner)
```

### Common Debug Commands

```bash
# Check runner status
tart-vm ssh plur-runner "launchctl list | grep circleci"

# View runner logs
tart-vm ssh plur-runner "tail -50 ~/Library/Logs/com.circleci.runner/runner.log"

# Check tool versions
tart-vm ssh plur-runner "source ~/.zshrc && ruby --version && go version"

# Restart the runner
tart-vm ssh plur-runner "launchctl unload ~/Library/LaunchAgents/com.circleci.runner.plist && launchctl load ~/Library/LaunchAgents/com.circleci.runner.plist"
```

## Troubleshooting

### VM Won't Start (Keychain Error)

macOS 15+ requires an unlocked keychain to run VMs. Either:

1. Connect via Screen Sharing once to initialize the GUI session, or
2. Unlock the keychain:
   ```bash
   security unlock-keychain -p '' login.keychain
   ```

### SSH Connection Refused

Enable Remote Login on the VM:
System Settings → General → Sharing → Remote Login → On

### VM IP Changes

The VM gets a dynamic IP via DHCP. Always use `tart ip plur-runner` to get the current address. The `tart-vm ssh` and `tart-vm scp` commands handle IP lookup automatically.

### Runner Not Claiming Jobs

1. Check the runner is running:
   ```bash
   tart-vm ssh plur-runner "launchctl list | grep circleci"
   ```

2. Check logs for errors:
   ```bash
   tart-vm ssh plur-runner "tail -100 ~/Library/Logs/com.circleci.runner/runner.log"
   ```

3. Verify the auth token is valid (regenerate from 1Password if needed)

### Provisioning a Fresh VM

If you need to rebuild the VM from scratch:

```bash
# Clone fresh base image
tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest plur-runner

# Configure resources
tart set plur-runner --cpu 6 --memory 16384 --disk-size 60

# Start VM (opens GUI window, already logged in)
tart run plur-runner
```

The Cirrus Labs base images come pre-configured—no Setup Assistant or login required. The VM opens already logged in as `admin`.

**In the VM GUI:**

1. Enable Remote Login: System Settings → General → Sharing → Remote Login → On

**Set up SSH key access from the host:**

```bash
# Generate a key if you don't have one
ssh-keygen -t ed25519 -f ~/.ssh/ci-vm-key -N ''

# Copy public key to VM (will prompt for password: "admin")
ssh-copy-id -i ~/.ssh/ci-vm-key admin@$(tart ip plur-runner)
```

**Install development tools:**

```bash
# SSH in
tart-vm ssh plur-runner

# Inside VM - install base tooling first (if fresh VM)
vm-bootstrap

# Then install plur-specific tools
cd /path/to/plur && script/ci-host-setup
```

The CircleCI runner setup (token, LaunchAgent) is a separate step—see [self-hosted-ci-implementation.md](../wip/self-hosted-ci-implementation.md) for details.

## References

* [Tart Documentation](https://tart.run/)
* [Tart Quick Start](https://tart.run/quick-start/)
* [Planning Doc](../wip/self-hosted-ci-implementation.md) - Full implementation plan
* [Security Research](../wip/self-hosted-ci.md) - Why VM isolation matters
* [CircleCI Machine Runner](https://circleci.com/docs/install-machine-runner-3-on-macos/)