# Self-Hosted CI Implementation Plan

*Status:* In Progress
*Created:* 2026-01-14
*Related:* [self-hosted-ci.md](self-hosted-ci.md) (research & reference)

## Overview

Implement a secure, VM-isolated CircleCI self-hosted runner using Tart on Mac Studio. The runner will execute inside a macOS VM, providing hypervisor-level isolation from the host.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Mac Studio (Host)                                           │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Tart VM: plur-runner                                  │  │
│  │ (macOS Sequoia, 4 CPU, 8GB RAM)                       │  │
│  │                                                       │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │ CircleCI Machine Runner                         │  │  │
│  │  │ * Resource class: rsanheim/mac-studio           │  │  │
│  │  │ * Token via 1Password (op read)                 │  │  │
│  │  │ * Runs as LaunchAgent                           │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  │                                                       │  │
│  │  Tools: mise → ruby 4, go 1.25, python 3              │  │
│  │  Plur: cloned from git, built with bin/rake install   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  Host only runs: Tart (VM management)                       │
└─────────────────────────────────────────────────────────────┘
```

## Success Criteria

* [ ] Tart VM boots and is accessible via SSH from host
* [ ] VM has mise, Ruby 4, Go 1.25, Python 3 installed and working
* [ ] CircleCI machine runner runs inside VM and claims jobs
* [ ] Plur builds and tests pass when triggered from CircleCI
* [ ] VM startup is automated (host launchd or manual script)
* [ ] Setup is documented and reproducible

## Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| VM persistence | Long-running (not ephemeral) | Simpler for v1; ephemeral can be added later |
| Tool installation | mise (not brew/manual) | Consistent with host setup, easy version management |
| Token storage | Config file in VM | Simple; token retrieved from host's 1Password during setup |
| Runner mode | Continuous (not single-task) | Standard runner behavior |

## Phase 1: Tart VM Base Setup

**Goal**: Get a macOS VM running and accessible via SSH from the host.

### Prerequisites (on Host)

* [ ] Tart installed: `brew install cirruslabs/cli/tart`
* [ ] Sufficient disk space (~25GB for VM image)
* [ ] 1Password CLI installed and authenticated

### Tasks

* [ ] Clone base macOS image
  ```bash
  tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest plur-runner
  ```

* [ ] Configure VM resources
  ```bash
  tart set plur-runner --cpu 4 --memory 8192
  ```

* [ ] Start VM and complete initial setup
  ```bash
  tart run plur-runner
  ```
  * Default credentials: `admin` / `admin`
  * Enable Remote Login (SSH) in System Settings → General → Sharing
  * Optionally set up auto-login

* [ ] Verify SSH access from host
  ```bash
  ssh admin@$(tart ip plur-runner) "uname -a && sw_vers"
  ```

* [ ] Add SSH key for passwordless access (optional but recommended)
  ```bash
  ssh-copy-id admin@$(tart ip plur-runner)
  ```

* [ ] Install Homebrew in VM (if not already present)
  ```bash
  ssh admin@$(tart ip plur-runner)
  # Check if brew exists
  which brew || /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  # Add to PATH if needed
  echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
  eval "$(/opt/homebrew/bin/brew shellenv)"
  ```

### Validation

```bash
# From host - verify SSH and Homebrew
ssh admin@$(tart ip plur-runner) "echo 'VM accessible!' && brew --version"
```

## Phase 2: VM Development Environment

**Goal**: Install mise, Ruby, Go, Python, and Bundler inside the VM.

### Automated Setup

Use `script/ci-host-setup` to install all development tools:

```bash
# From host - clone plur repo and run setup script
ssh admin@$(tart ip plur-runner) "git clone https://github.com/rsanheim/plur.git ~/plur"
ssh admin@$(tart ip plur-runner) "cd ~/plur && script/ci-host-setup"
```

The script handles:
* Xcode Command Line Tools
* Homebrew
* mise (version manager)
* Ruby 4, Go 1.25, Python 3 (via mise, from `.mise.toml`)
* Bundler

### Manual Alternative

If you prefer manual setup or need to debug:

```bash
ssh admin@$(tart ip plur-runner)

# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
eval "$(/opt/homebrew/bin/brew shellenv)"

# Install mise
brew install mise
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc
source ~/.zshrc

# Install tools from .mise.toml
cd ~/plur
mise trust && mise install

# Install bundler
gem install bundler
```

### Validation

```bash
# From host - verify all tools
ssh admin@$(tart ip plur-runner) "ruby --version && go version && python --version && bundler --version"
# Expected: ruby 4.x, go 1.25.x, python 3.x, bundler 4.x
```

## Phase 3: CircleCI Runner in VM

**Goal**: Install and configure CircleCI machine runner inside the VM.

### Tasks

* [ ] Install runner via Homebrew (inside VM)
  ```bash
  ssh admin@$(tart ip plur-runner)
  # Then inside VM:
  brew tap circleci-public/circleci
  brew install circleci-runner
  ```

* [ ] Clear quarantine attribute
  ```bash
  sudo xattr -r -d com.apple.quarantine "$(brew --prefix)/bin/circleci-runner"
  ```

* [ ] Get token from host's 1Password and create config (run from host)
  ```bash
  # From HOST - get token and write config to VM
  TOKEN=$(op read 'op://Private/circle ci self hosted runner/credential')

  ssh admin@$(tart ip plur-runner) "mkdir -p ~/Library/Preferences/com.circleci.runner"

  ssh admin@$(tart ip plur-runner) "cat > ~/Library/Preferences/com.circleci.runner/config.yaml << EOF
  runner:
    name: plur-runner-vm
    working_directory: /Users/admin/Library/com.circleci.runner/workdir
    cleanup_working_directory: true
  api:
    auth_token: $TOKEN
  EOF"
  ```

* [ ] Test runner starts manually (inside VM)
  ```bash
  circleci-runner machine --config ~/Library/Preferences/com.circleci.runner/config.yaml
  # Should connect and show "waiting for task"
  # Ctrl-C to stop
  ```

* [ ] Set up LaunchAgent for auto-start (inside VM)
  ```bash
  cat > ~/Library/LaunchAgents/com.circleci.runner.plist << 'EOF'
  <?xml version="1.0" encoding="UTF-8"?>
  <!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
  <plist version="1.0">
  <dict>
      <key>Label</key>
      <string>com.circleci.runner</string>
      <key>Program</key>
      <string>/opt/homebrew/bin/circleci-runner</string>
      <key>ProgramArguments</key>
      <array>
          <string>/opt/homebrew/bin/circleci-runner</string>
          <string>machine</string>
          <string>--config</string>
          <string>/Users/admin/Library/Preferences/com.circleci.runner/config.yaml</string>
      </array>
      <key>RunAtLoad</key>
      <true/>
      <key>KeepAlive</key>
      <true/>
      <key>StandardOutPath</key>
      <string>/Users/admin/Library/Logs/com.circleci.runner/runner.log</string>
      <key>StandardErrorPath</key>
      <string>/Users/admin/Library/Logs/com.circleci.runner/runner.log</string>
  </dict>
  </plist>
  EOF

  mkdir -p ~/Library/Logs/com.circleci.runner
  launchctl load ~/Library/LaunchAgents/com.circleci.runner.plist
  ```

* [ ] Verify runner is running
  ```bash
  launchctl list | grep circleci
  tail -f ~/Library/Logs/com.circleci.runner/runner.log
  ```

### Validation

```bash
# From host - check runner is connected
ssh admin@$(tart ip plur-runner) "launchctl list | grep circleci"
```

## Phase 4: CircleCI Workflow Update

**Goal**: Update CircleCI config to run real Plur tests on the VM runner.

### Tasks

* [ ] Update `.circleci/config.yml` - modify `hello-macos-arm64` job
  ```yaml
  hello-macos-arm64:
    machine: true
    resource_class: rsanheim/mac-studio
    steps:
      - checkout
      - run:
          name: Environment info
          command: |
            echo "Running in Tart VM"
            uname -a
            sw_vers
            ruby --version
            go version
      - run:
          name: Install dependencies
          command: |
            bundle install
      - run:
          name: Build plur
          command: |
            bin/rake build
      - run:
          name: Run tests
          command: |
            ./plur/plur --version
            ./plur/plur doctor
            bin/rake test:go test:default_ruby
  ```

* [ ] Commit and push to trigger workflow
  ```bash
  git add .circleci/config.yml
  git commit -m "Update self-hosted runner job to run Plur tests"
  git push
  ```

* [ ] Monitor job execution
  * Watch CircleCI UI for job status
  * Check VM runner logs: `ssh admin@$(tart ip plur-runner) "tail -f ~/Library/Logs/com.circleci.runner/runner.log"`

### Validation

* [ ] Job runs successfully on self-hosted runner
* [ ] All tests pass
* [ ] Job completes in reasonable time

## Phase 5: Host Automation (Optional)

**Goal**: Automate VM startup from the host.

### Option A: Manual Start Script

```bash
# ~/bin/start-plur-runner-vm
#!/bin/bash
set -e

VM_NAME="plur-runner"

# Check if VM is already running
if tart list | grep -q "$VM_NAME.*running"; then
    echo "VM $VM_NAME is already running"
    exit 0
fi

echo "Starting $VM_NAME..."
tart run --no-graphics "$VM_NAME" &

# Wait for VM to be accessible
echo "Waiting for SSH..."
for i in {1..60}; do
    if ssh -o ConnectTimeout=2 -o BatchMode=yes admin@$(tart ip "$VM_NAME" 2>/dev/null) "exit" 2>/dev/null; then
        echo "VM is ready!"
        exit 0
    fi
    sleep 1
done

echo "Timeout waiting for VM"
exit 1
```

### Option B: Host LaunchAgent

Create a LaunchAgent on the host that starts the VM at login.

### Tasks

* [ ] Choose approach (manual script vs LaunchAgent)
* [ ] Implement chosen approach
* [ ] Test VM auto-starts after host reboot

## Phase 6: Cleanup Host Runner (Optional)

**Goal**: Remove or disable the unisolated runner on the host.

### Tasks

* [ ] Stop host runner
  ```bash
  launchctl unload ~/Library/LaunchAgents/com.circleci.runner.plist
  ```

* [ ] Optionally remove host runner config
  ```bash
  rm -rf ~/Library/Preferences/com.circleci.runner
  rm ~/Library/LaunchAgents/com.circleci.runner.plist
  rm ~/.local/bin/circleci-runner-wrapper
  ```

* [ ] Or keep for fallback/testing

## Troubleshooting

### VM Won't Start (Keychain Error)

macOS 15+ requires unlocked keychain. Connect via Screen Sharing first, or:

```bash
security create-keychain -p '' login.keychain
security unlock-keychain -p '' login.keychain
```

### Runner Can't Authenticate

Verify the token in the config file is correct:

```bash
ssh admin@$(tart ip plur-runner) "head -10 ~/Library/Preferences/com.circleci.runner/config.yaml"
```

If token is wrong/expired, regenerate from host:

```bash
TOKEN=$(op read 'op://Private/circle ci self hosted runner/credential')
ssh admin@$(tart ip plur-runner) "sed -i '' \"s/auth_token:.*/auth_token: $TOKEN/\" ~/Library/Preferences/com.circleci.runner/config.yaml"
```

### SSH Connection Refused

Enable Remote Login in VM:
System Settings → General → Sharing → Remote Login → On

### VM IP Changes

Tart assigns IPs dynamically. Use `tart ip plur-runner` to get current IP.

For stable access, consider:
* Adding VM hostname to `/etc/hosts` (update on IP change)
* Using mDNS if available

## Files Created/Modified

| Location | File | Purpose |
|----------|------|---------|
| Repo | `script/ci-host-setup` | Automated VM setup script |
| Repo | `.mise.toml` | Tool versions (Ruby 4, Go 1.25, Python 3) |
| Host | `~/bin/start-plur-runner-vm` | VM startup script (optional) |
| VM | `~/Library/Preferences/com.circleci.runner/config.yaml` | Runner config (includes token) |
| VM | `~/Library/LaunchAgents/com.circleci.runner.plist` | Runner auto-start |
| Repo | `.circleci/config.yml` | Updated workflow |

## References

* [self-hosted-ci.md](self-hosted-ci.md) - Research and security model details
* [Tart Quick Start](https://tart.run/quick-start/)
* [CircleCI Machine Runner Installation](https://circleci.com/docs/install-machine-runner-3-on-macos/)
* [mise Getting Started](https://mise.jdx.dev/getting-started.html)
