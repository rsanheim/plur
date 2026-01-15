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
| Resources | 4 CPU, 8GB RAM |
| User | `admin` |
| SSH Key | `~/.ssh/ci-vm-key` |

## Managing the VM

### Tart Commands

```bash
# List all VMs and their status
tart list

# Start VM (headless, backgrounded)
tart run --no-graphics plur-runner &

# Start VM with GUI window (useful for debugging)
tart run plur-runner

# Stop VM gracefully
tart stop plur-runner

# Get current VM IP address
tart ip plur-runner

# Configure VM resources
tart set plur-runner --cpu 4 --memory 8192

# Clone a new VM from registry
tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest my-new-vm
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

### Helper Scripts

We have wrapper scripts that handle IP lookup and key management:

**script/ci-vm-ssh** - SSH into the VM:

```bash
script/ci-vm-ssh                    # Interactive shell
script/ci-vm-ssh "uname -a"         # Run a command
script/ci-vm-ssh -t "top"           # With TTY (for interactive commands)
```

**script/ci-vm-scp** - Copy files to/from VM:

```bash
script/ci-vm-scp local-file.txt :~/remote-path/    # Copy TO VM
script/ci-vm-scp :~/remote-file.txt ./local-path/  # Copy FROM VM
script/ci-vm-scp -r local-dir :~/                  # Recursive copy
```

The `:` prefix indicates "on the VM" and gets expanded to `admin@<vm-ip>:`.

### Direct SSH

If the helper scripts aren't available:

```bash
ssh -i ~/.ssh/ci-vm-key admin@$(tart ip plur-runner)
```

### Common Debug Commands

```bash
# Check runner status
script/ci-vm-ssh "launchctl list | grep circleci"

# View runner logs
script/ci-vm-ssh "tail -50 ~/Library/Logs/com.circleci.runner/runner.log"

# Check tool versions
script/ci-vm-ssh "source ~/.zshrc && ruby --version && go version"

# Restart the runner
script/ci-vm-ssh "launchctl unload ~/Library/LaunchAgents/com.circleci.runner.plist && launchctl load ~/Library/LaunchAgents/com.circleci.runner.plist"
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

The VM gets a dynamic IP via DHCP. Always use `tart ip plur-runner` to get the current address. The helper scripts handle this automatically.

### Runner Not Claiming Jobs

1. Check the runner is running:
   ```bash
   script/ci-vm-ssh "launchctl list | grep circleci"
   ```

2. Check logs for errors:
   ```bash
   script/ci-vm-ssh "tail -100 ~/Library/Logs/com.circleci.runner/runner.log"
   ```

3. Verify the auth token is valid (regenerate from 1Password if needed)

### Provisioning a Fresh VM

If you need to rebuild the VM from scratch:

```bash
# Clone fresh base image
tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest plur-runner

# Configure resources
tart set plur-runner --cpu 4 --memory 8192

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
# SSH in and run setup
script/ci-vm-ssh
# Inside VM:
cd /path/to/plur && script/ci-host-setup
```

The CircleCI runner setup (token, LaunchAgent) is a separate step—see [self-hosted-ci-implementation.md](../wip/self-hosted-ci-implementation.md) for details.

## References

* [Tart Documentation](https://tart.run/)
* [Tart Quick Start](https://tart.run/quick-start/)
* [Planning Doc](../wip/self-hosted-ci-implementation.md) - Full implementation plan
* [Security Research](../wip/self-hosted-ci.md) - Why VM isolation matters
* [CircleCI Machine Runner](https://circleci.com/docs/install-machine-runner-3-on-macos/)
