The below test cases demonstrate various bugs with 'plur watch' output -- they should
be considered together for the fixes, as the newline and logging issues are inter-dependent.

* S1 - the 'rspec ....' line should be on the same line as the 'plur' prompt, as if it were a command.  
We have a newline somewhere here.

* S2 - this is when run in debug mode -- note that the DEBUG output interleaves on the prompt, and then 
we have the command run w/ [plur] prefix (not a proper prompt for some reason).

* S3 - this is a failuring spec -- note that we get two WARN messages down below the rspec error output,
which is redundant. If we _do_ want to log the exit code from the plur spawned command, we should probably
do it at INFO level (as default log level is WARN I think?), and only do it _once_.


### S1

```
[plur] >
[plur] rspec spec/lib/dx/models/package_info_formatter_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 43258

Dx::Models::PackageInfoFormatter
  #format
    with a JavaScript package
      formats all fields with proper alignment
    with a gem package
      formats all fields with proper alignment
      formats each field on its own line
    with minimal package (missing optional fields)
      only shows fields that are present
      has 4 lines (name, version, type, location)

Finished in 0.01705 seconds (files took 0.12852 seconds to load)
5 examples, 0 failures

Randomized with seed 43258
```

### S2

```
[plur] > 14:06:41 - DEBUG - watch path="spec/lib/dx/models/package_info_formatter_spec.rb" fullPath="/Users/rsanheim/work/gems/dox-dx-ruby/spec/lib/dx/models/package_info_formatter_spec.rb" event="modify" type="file"
14:06:41 - DEBUG - renderTargets result normalizedPath="spec/lib/dx/models/package_info_formatter_spec.rb" watch="spec/**/*.rb" targets=[spec/lib/dx/models/package_info_formatter_spec.rb]
14:06:41 - INFO  - Executing job job="rspec" targets="[spec/lib/dx/models/package_info_formatter_spec.rb]"

[plur] rspec spec/lib/dx/models/package_info_formatter_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 48623

Dx::Models::PackageInfoFormatter
  #format
    with a gem package
      formats all fields with proper alignment
      formats each field on its own line
    with minimal package (missing optional fields)
      has 4 lines (name, version, type, location)
      only shows fields that are present
    with a JavaScript package
      formats all fields with proper alignment

Finished in 0.01833 seconds (files took 0.13204 seconds to load)
5 examples, 0 failures

Randomized with seed 48623


[plur] >
```

### S3

```
[plur] >
[plur] rspec spec/lib/dx/environment_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 24155

Dx::Environment
  empty and nil are invalid
  works with pre / pre-prod

Failures:

  1) Dx::Environment Environment::Null returns false for all env checks
     Failure/Error: fail
     RuntimeError:
     # ./spec/lib/dx/environment_spec.rb:19:in 'block (3 levels) in <top (required)>'
     # ./spec/spec_helper.rb:63:in 'block (2 levels) in <top (required)>'

Finished in 0.02127 seconds (files took 0.14032 seconds to load)
19 examples, 1 failure

Failed examples:

rspec ./spec/lib/dx/environment_spec.rb:17 # Dx::Environment Environment::Null returns false for all env checks

Randomized with seed 24155

14:10:58 - WARN  - Job execution failed job="rspec" error=exit status 1
14:10:58 - WARN  - Job execution error job="rspec" error=exit status 1
```




