require "spec_helper"

RSpec.describe "CLI inventory harness" do
  let(:script) { ROOT_PATH.join("script", "cli-inventory") }

  def run_inventory(env = {})
    binary = plur_binary
    binary = File.expand_path(binary) if binary.include?(File::SEPARATOR)

    Open3.capture3(
      env.merge("PLUR_BINARY" => binary),
      script.to_s,
      chdir: ROOT_PATH.to_s
    )
  end

  it "lists every cli_inventory command in repeatable summary mode" do
    stdout, stderr, status = run_inventory

    expect(status).to be_success, stderr
    expect(stdout).to include("CLI inventory")
    [
      "plur",
      "plur spec",
      "plur spec foo_spec.rb",
      "plur test",
      "plur test foo_test.rb",
      "plur spec/**/*.rb",
      "plur --use custom-job",
      "plur foo/baz/other-file.go",
      "plur foo/baz/other_test.rs",
      "plur foo_spec.rb bar_spec.rb",
      "plur -C ~/src/oss/rubocoop spec",
      "plur spec --exclude-pattern '*user*/_spec.rb'",
      "plur foo/**/*_spec.rb other/**/*_spec.rb",
      "plur --help",
      "plur help spec",
      "plur \"foo/(1|2|3|)_spec.rb\"",
      "plur watch"
    ].each do |label|
      expect(stdout).to include(label)
    end
  end

  it "prints intentions and command output in PLUR_DEMO mode" do
    stdout, stderr, status = run_inventory("PLUR_DEMO" => "1")

    expect(status).to be_success, stderr
    expect(stdout).to include("INTENT:")
    expect(stdout).to include("COMMAND:")
    expect(stdout).to include("STDERR:")
    expect(stdout).to include("[dry-run] Running")
    expect(stdout).to include("plur watch")
  end

  it "can run summary and demo inventories concurrently" do
    results = 24.times.map do |index|
      Thread.new do
        run_inventory(index.even? ? {} : {"PLUR_DEMO" => "1"})
      end
    end.map(&:value)

    results.each do |stdout, stderr, status|
      expect(status).to be_success, stderr
      expect(stdout).to include("CLI inventory")
    end
  end
end
