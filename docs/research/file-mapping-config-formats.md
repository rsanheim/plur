# File Mapping Configuration Formats

## Overview
Comparison of different configuration formats for managing file-to-spec mappings in plur watch.

## 1. YAML Format (Recommended)
**Pros**: Human-readable, widely supported, comments, multi-line strings
**Cons**: Indentation-sensitive

```yaml
# .plur-watch.yml
version: 1
mappings:
  # Simple glob patterns
  - pattern: "lib/**/*.rb"
    specs: "spec/**/*_spec.rb"
  
  # Rails-style mappings
  - pattern: "app/models/*.rb"
    specs: "spec/models/{}_spec.rb"
  
  - pattern: "app/controllers/*_controller.rb"
    specs: 
      - "spec/controllers/{}_spec.rb"
      - "spec/requests/{}_spec.rb"
  
  # Special files that run all specs
  - pattern: "spec/spec_helper.rb"
    specs: "spec/**/*_spec.rb"
  
  # Regex patterns with captures
  - pattern: "app/(.*).rb"
    regex: true
    specs: "spec/$1_spec.rb"
  
  # Custom transformations
  - pattern: "lib/foo/(.*)/(.*).rb"
    regex: true
    transform: 
      - "spec/foo/$1/${2}_spec.rb"
      - "spec/integration/foo_$1_spec.rb"

# Global settings
settings:
  default_spec_dir: "spec"
  run_all_pattern: "spec/**/*_spec.rb"
  debounce: 100
```

## 2. JSON Format
**Pros**: Native Go support, no external deps, precise
**Cons**: No comments, verbose, harder to read

```json
{
  "version": 1,
  "mappings": [
    {
      "pattern": "lib/**/*.rb",
      "specs": ["spec/**/*_spec.rb"]
    },
    {
      "pattern": "app/models/*.rb",
      "specs": ["spec/models/{}_spec.rb"]
    },
    {
      "pattern": "app/(.*).rb",
      "regex": true,
      "specs": ["spec/$1_spec.rb"]
    }
  ],
  "settings": {
    "defaultSpecDir": "spec",
    "runAllPattern": "spec/**/*_spec.rb",
    "debounce": 100
  }
}
```

## 3. TOML Format
**Pros**: Clean syntax, comments, good for configs
**Cons**: Less common in Ruby ecosystem

```toml
# .plur-watch.toml
version = 1

[settings]
default_spec_dir = "spec"
run_all_pattern = "spec/**/*_spec.rb"
debounce = 100

[[mappings]]
pattern = "lib/**/*.rb"
specs = ["spec/**/*_spec.rb"]

[[mappings]]
pattern = "app/models/*.rb"
specs = ["spec/models/{}_spec.rb"]

[[mappings]]
pattern = "app/(.*).rb"
regex = true
specs = ["spec/$1_spec.rb"]
```

## 4. Ruby DSL (Like Guardfile)
**Pros**: Familiar to Ruby devs, powerful, can use Ruby logic
**Cons**: Requires Ruby parser in Go, security concerns

```ruby
# .plur-watch.rb or Plurfile
watch('lib/**/*.rb') do |m|
  "spec/#{m[1]}_spec.rb"
end

watch('app/models/(.*)\.rb') do |m|
  "spec/models/#{m[1]}_spec.rb"
end

watch('app/controllers/(.*)_controller\.rb') do |m|
  [
    "spec/controllers/#{m[1]}_controller_spec.rb",
    "spec/requests/#{m[1]}_spec.rb"
  ]
end

# Special files
watch('spec/spec_helper.rb') { 'spec' }
watch('Gemfile') { 'spec' }

# With options
watch('app/views/(.*)\.html\.erb', run_all: true) do |m|
  "spec/views/#{m[1]}_spec.rb"
end
```

## 5. Lua Script (Embedded)
**Pros**: Fast, sandboxed, programmable
**Cons**: Another language to learn, requires Lua interpreter

```lua
-- .plur-watch.lua
mappings = {
  {
    pattern = "lib/(.*).rb",
    transform = function(matches)
      return "spec/" .. matches[1] .. "_spec.rb"
    end
  },
  {
    pattern = "app/models/(.*).rb",
    transform = function(matches)
      return "spec/models/" .. matches[1] .. "_spec.rb"
    end
  }
}

-- Special handling
special_files = {
  ["spec/spec_helper.rb"] = "spec/**/*_spec.rb",
  ["Gemfile"] = "spec/**/*_spec.rb"
}
```

## Translation Examples

### Guardfile to YAML
```ruby
# Guardfile
watch(%r{^lib/(.+)\.rb$}) { |m| "spec/lib/#{m[1]}_spec.rb" }
```

```yaml
# .plur-watch.yml
- pattern: "lib/(.+).rb"
  regex: true
  specs: "spec/lib/$1_spec.rb"
```

### YAML to Go Struct
```go
type Mapping struct {
    Pattern   string   `yaml:"pattern"`
    Regex     bool     `yaml:"regex"`
    Specs     []string `yaml:"specs"`
    Transform []string `yaml:"transform"`
}

type Config struct {
    Version  int                    `yaml:"version"`
    Mappings []Mapping              `yaml:"mappings"`
    Settings map[string]interface{} `yaml:"settings"`
}
```

## Recommendation

**YAML** is the best choice because:
1. Human-readable and editable
2. Supports comments for documentation
3. Well-supported in Go (gopkg.in/yaml.v3)
4. Common in DevOps/CI tools
5. Can express complex mappings clearly
6. Easy to translate to/from Guardfile patterns

The format supports:
- Glob patterns (lib/**/*.rb)
- Regex patterns with capture groups
- Multiple spec targets per pattern
- Placeholder substitution ({} or $1 style)
- Special file handling
- Global settings

Implementation would involve:
1. Parse YAML config
2. Compile patterns (glob or regex)
3. Match changed files against patterns
4. Apply transformations to generate spec paths
5. Cache compiled patterns for performance