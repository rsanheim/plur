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

Doctor can report missing tools or job selection errors and still exit
successfully. Use its output to troubleshoot CI failures; its exit status does
not tell you whether the environment is ready to run tests.

For debug logging during configuration loading:

```bash
PLUR_DEBUG=1 plur doctor
```
