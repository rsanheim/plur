# symlinks

Real symlinks for exercising how plur resolves links when detecting and
collecting spec files. `spec/integration/spec/symlinks_spec.rb` runs
`plur spec` in each project root below. The Gemfile reuses the root Gemfile
through `eval_gemfile`, and `Gemfile.lock` is a symlink to the root lockfile,
so the projects run for real without duplicating gem declarations.

Project roots:

* `app/` - links inside `spec/` and `lib/`
  * `spec/linked-outside -> ../../shared/spec` (directory, real path outside the project)
  * `spec/linked-inside -> ../lib/specs` (directory, real path inside the project)
  * `spec/linked_file_spec.rb -> ../../shared/spec/remote_spec.rb` (file, real path outside the project)
  * `lib/vendored -> ../../other-project` (another project with its own `spec/`)
* `app-link -> app` - the project root itself reached through a link
* `spec-link-inside/` - `spec -> real-spec`, the spec directory is a link to a real directory inside the project
* `spec-link-outside/` - `spec -> ../shared/spec`, the spec directory is a link to a directory outside the project

Link targets:

* `shared/spec/remote_spec.rb`
* `other-project/spec/other_spec.rb`
