# Doctor Command

`plur doctor` prints installation, environment, and configuration details for
troubleshooting:

```bash
plur doctor
```

## What It Reports

* Plur version, build metadata, binary path, and race detector status
* Operating system, architecture, CPU count, Go version, and working directory
* Ruby, Bundler, and RSpec versions or availability errors
* Watcher availability, binary path, and version
* Cache directory and runtime data path, with file size and test counts when available
* Environment variables, active configuration files, worker count, and color settings
* Selected job, command, target patterns, and existing watch directories

Doctor prints diagnostic findings; missing tools or an unavailable job can appear
in its output without causing a nonzero exit status. In CI, use it to collect
troubleshooting information rather than as a pass/fail environment check.

For debug logging during configuration loading:

```bash
PLUR_DEBUG=1 plur doctor
```
