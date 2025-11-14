require "spec_helper"

RSpec.describe "plur watch find with mismatched directory structure" do
  let(:fixture_dir) { project_fixture("rspec-mismatched-dirs") }

  it "shows matched rules and missing targets for mismatched structure" do
    cmd = TTY::Command.new(uuid: false, printer: :null)

    # Run watch find - exits with code 2 (no executable targets)
    result = nil
    begin
      result = cmd.run!("plur watch find lib/example/runner.rb", chdir: fixture_dir)
    rescue TTY::Command::ExitError => e
      result = e.result
    end

    # Should show checking message in slog format
    expect(result.out).to include('msg="checking watch" file=lib/example/runner.rb')

    # Should show matched rules in slog format
    expect(result.out).to include('msg="found rules" name=lib-to-spec source=lib/**/*.rb')
    expect(result.out).to include("target=spec/{{match}}_spec.rb")

    # Should show missing files as warnings
    expect(result.out).to include('msg="not found" file=spec/example/runner_spec.rb')

    # Should exit with code 2 (no targets found)
    expect(result.exit_status).to eq(2)

    # Verify the files actually don't exist / do exist at wrong location
    expected_file = fixture_dir.join("spec/example/runner_spec.rb")
    expect(expected_file.exist?).to be false

    actual_file = fixture_dir.join("spec/lib/example/runner_spec.rb")
    expect(actual_file.exist?).to be true
  end

  it "shows executable targets for files with correct structure" do
    cmd = TTY::Command.new(uuid: false, printer: :null)
    result = cmd.run!("plur watch find lib/example.rb", chdir: fixture_dir)

    # Should show checking message in slog format
    expect(result.out).to include('msg="checking watch" file=lib/example.rb')

    # Should show matched rules in slog format
    expect(result.out).to include('msg="found rules" name=lib-to-spec')

    # Should show found files in slog format
    expect(result.out).to include('msg="found files" files=spec/example_spec.rb')

    # Should exit with code 0 (found targets)
    expect(result.exit_status).to eq(0)
  end

  it "shows both executable and missing targets when mixed" do
    cmd = TTY::Command.new(uuid: false, printer: :null)
    result = cmd.run!("plur watch find lib/example.rb", chdir: fixture_dir)

    # Should show found files in slog format
    expect(result.out).to include('msg="found files" files=spec/example_spec.rb')

    # Should also show missing files as warnings
    expect(result.out).to include('msg="not found" file=test/example_test.rb')

    # Exit code 0 because at least one target exists
    expect(result.exit_status).to eq(0)
  end
end
