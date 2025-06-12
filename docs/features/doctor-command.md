# Doctor Command

The `rux doctor` command helps diagnose installation and configuration issues.

## Usage

```bash
rux doctor
```

## What It Checks

1. **Rux Installation**
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

## Common Issues

### "rux: command not found"
- Run `bin/rake install` from project root
- Check `$GOPATH/bin` is in your PATH

### "cannot load such file -- backspin"
- Run `bundle install` at project root
- Ensure you're in a bundled environment

### "no test files found"
- Check for `*_spec.rb` files
- Verify working directory
- Check file permissions

## Debug Output

For verbose diagnostics:
```bash
RUX_DEBUG=1 rux doctor
```

## Integration with CI

Use doctor in CI to validate environment:
```yaml
# GitHub Actions
- name: Validate Rux setup
  run: rux doctor
```