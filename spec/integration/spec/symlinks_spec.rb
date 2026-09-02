require "spec_helper"

# Runs `plur spec` inside fixtures/projects/symlinks, which holds real symlinks
# (see its README). Each row is a path as it appears in the project, what it
# really points at, and the files `plur spec` collects through it. Collection
# follows links wherever they lead; the project root and the spec directory
# resolve through the OS like any other path.
RSpec.describe "plur spec with symlinks" do
  fixture = project_fixture("symlinks")
  cases = {
    "app" => [
      {target: "spec/local_spec.rb", real: "app/spec/local_spec.rb", collect: ["spec/local_spec.rb"]},
      {target: "spec/linked-outside", real: "shared/spec", collect: ["spec/linked-outside/remote_spec.rb"]},
      {target: "spec/linked-inside", real: "app/lib/specs", collect: ["spec/linked-inside/lib_spec.rb"]},
      {target: "spec/linked_file_spec.rb", real: "shared/spec/remote_spec.rb", collect: ["spec/linked_file_spec.rb"]},
      {target: "lib/vendored", real: "other-project", collect: []}
    ],
    "app-link" => [
      {target: ".", real: "app", collect: [
        "spec/linked-inside/lib_spec.rb", "spec/linked-outside/remote_spec.rb", "spec/linked_file_spec.rb", "spec/local_spec.rb"
      ]}
    ],
    "spec-link-inside" => [
      {target: "spec", real: "spec-link-inside/real-spec", collect: ["spec/local_spec.rb"]}
    ],
    "spec-link-outside" => [
      {target: "spec", real: "shared/spec", collect: ["spec/remote_spec.rb"]}
    ]
  }

  # Runs plur under the fixture's own Gemfile rather than the one the outer
  # Bundler exported, so these examples prove the fixture runs standalone.
  def run_standalone(dir)
    Bundler.with_unbundled_env do
      Dir.chdir(dir) do
        _, _, status = Open3.capture3("bundle", "check")
        Open3.capture3("bundle", "install", "--quiet") unless status.success?
      end
      run_plur("-C", dir, "-n", "1", "spec")
    end
  end

  def collected_files(dir)
    result = run_plur("-C", dir, "-n", "1", "--dry-run", "spec")
    worker = result.err.lines.find { |line| line.include?("[dry-run] Worker 0:") }
    expect(worker).not_to be_nil, "no dry-run worker line in:\n#{result.err}"
    worker.scan(/\S+_spec\.rb/).sort
  end

  cases.each do |project, rows|
    describe "in #{project}" do
      rows.each do |row|
        expected = row[:collect].empty? ? "nothing" : row[:collect].join(", ")

        it "#{row[:target]} really is #{row[:real]}" do
          expect(File.realpath(fixture.join(project, row[:target]))).to eq(File.realpath(fixture.join(row[:real])))
        end

        it "#{row[:target]} -> #{row[:real]} collects #{expected}" do
          collected = collected_files(fixture.join(project))

          if row[:collect].empty?
            expect(collected.grep(%r{\A#{Regexp.escape(row[:target])}/})).to be_empty
          else
            expect(collected).to include(*row[:collect])
          end
        end
      end

      it "collects exactly the files the rows account for" do
        expect(collected_files(fixture.join(project))).to eq(rows.flat_map { |row| row[:collect] }.sort)
      end

      it "runs the collected specs" do
        result = run_standalone(fixture.join(project))

        expect(result.out).to match(/\b#{rows.sum { |row| row[:collect].size }} examples?, 0 failures/)
      end
    end
  end
end
