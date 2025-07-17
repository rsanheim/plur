# Doctor Command

The `plur doctor` command helps diagnose installation and configuration issues.

## Usage

```bash
 plur doctor
```

## What It Checks

1. **Plur Installation**
   - Binary location and permissions
   - Version information
   - Build metadata

2. **Ruby Environment**
   - Ruby version
   - RSpec availability
   - Bundler configuration

3. **Project Structure**
   - Test file discovery
   - Directory permissions
   - Git repository status

4. **System Resources**
   - Available CPU cores
   - Memory statistics
   - Process limits

## Debug Output

For verbose diagnostics:
```bash
PLUR_DEBUG=1 plur doctor
```

## Integration with CI

Use doctor in CI to validate environment:
```yaml
# GitHub Actions
- name: Validate Plur setup
  run: plur doctor
```