# CircleCI Self-Hosted Runner Setup

Reference for setting up CircleCI Machine Runner 3.0 on macOS.

## Installation (Homebrew)

```bash
brew tap circleci-public/circleci
brew install circleci-runner
```

## Paths

* **Logs**: `~/Library/Logs/com.circleci.runner/`
* **Config**: `~/Library/Preferences/com.circleci.runner/config.yaml`
* **LaunchAgent**: `~/Library/LaunchAgents/com.circleci.runner.plist`

## Configuration

Create `~/Library/Preferences/com.circleci.runner/config.yaml`:

```yaml
runner:
  name: "mac-studio"
  working_directory: "/Users/rsanheim/Library/com.circleci.runner/workdir"
  cleanup_working_directory: true

api:
  auth_token: "YOUR_TOKEN_HERE"
```

## macOS Security

```bash
# Check notarization
spctl -a -vvv -t install "$(brew --prefix)/bin/circleci-runner"

# Accept notarization headlessly
sudo xattr -r -d com.apple.quarantine "$(brew --prefix)/bin/circleci-runner"
```

## Service Management

```bash
# Load/reload the LaunchAgent
PLIST=~/Library/LaunchAgents/com.circleci.runner.plist
launchctl load $PLIST || (launchctl unload $PLIST && launchctl load $PLIST)

# Start manually (for debugging)
circleci-runner machine --config ~/Library/Preferences/com.circleci.runner/config.yaml

# View logs
tail -f ~/Library/Logs/com.circleci.runner/runner.log
```

## Documentation

* [Runner Overview](https://circleci.com/docs/runner-overview/)
* [Self-Hosted Runner Changelog](https://circleci.com/changelog/self-hosted-runner/)

## Plur Resource Class

* `rsanheim/mac-studio` - Mac Studio ARM64 runner
