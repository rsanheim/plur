require "spec_helper"
require "open3"

RSpec.describe "rux dev:file_mapper comprehensive mappings", skip: "dev:file_mapper command removed with urfave/cli2" do
  def run_file_mapper(file)
    stdout, stderr, status = Open3.capture3("rux", "dev:file_mapper", file)
    raise "Command failed: #{stderr}" unless status.success?
    stdout.strip
  end

  # Data-driven test cases
  mappings = [
    # Basic lib mappings
    {file: "lib/user.rb", expected: ["spec/user_spec.rb"]},
    {file: "lib/models/user.rb", expected: ["spec/models/user_spec.rb"]},
    {file: "lib/validators/email_validator.rb", expected: ["spec/validators/email_validator_spec.rb"]},

    # Rails app mappings
    {file: "app/models/user.rb", expected: ["spec/models/user_spec.rb"]},
    {file: "app/controllers/users_controller.rb", expected: ["spec/controllers/users_controller_spec.rb"]},
    {file: "app/controllers/api/v1/users_controller.rb", expected: ["spec/controllers/api/v1/users_controller_spec.rb"]},
    {file: "app/services/user_creator.rb", expected: ["spec/services/user_creator_spec.rb"]},
    {file: "app/helpers/users_helper.rb", expected: ["spec/helpers/users_helper_spec.rb"]},
    {file: "app/mailers/user_mailer.rb", expected: ["spec/mailers/user_mailer_spec.rb"]},
    {file: "app/jobs/user_cleanup_job.rb", expected: ["spec/jobs/user_cleanup_job_spec.rb"]},
    {file: "app/workers/user_worker.rb", expected: ["spec/workers/user_worker_spec.rb"]},
    {file: "app/decorators/user_decorator.rb", expected: ["spec/decorators/user_decorator_spec.rb"]},
    {file: "app/presenters/user_presenter.rb", expected: ["spec/presenters/user_presenter_spec.rb"]},
    {file: "app/serializers/user_serializer.rb", expected: ["spec/serializers/user_serializer_spec.rb"]},
    {file: "app/policies/user_policy.rb", expected: ["spec/policies/user_policy_spec.rb"]},
    {file: "app/forms/user_form.rb", expected: ["spec/forms/user_form_spec.rb"]},
    {file: "app/validators/user_validator.rb", expected: ["spec/validators/user_validator_spec.rb"]},
    {file: "app/models/concerns/trackable.rb", expected: ["spec/models/concerns/trackable_spec.rb"]},
    {file: "app/controllers/concerns/authenticatable.rb", expected: ["spec/controllers/concerns/authenticatable_spec.rb"]},

    # Spec files map to themselves
    {file: "spec/models/user_spec.rb", expected: ["spec/models/user_spec.rb"]},
    {file: "spec/controllers/users_controller_spec.rb", expected: ["spec/controllers/users_controller_spec.rb"]},

    # Special cases
    {file: "spec/spec_helper.rb", expected: ["spec"]},
    {file: "spec/rails_helper.rb", expected: ["spec"]},

    # Non-Ruby files - no mapping
    {file: "app/views/users/index.html.erb", expected: []},
    {file: "app/assets/stylesheets/users.scss", expected: []},
    {file: "app/javascript/controllers/user_controller.js", expected: []},
    {file: "config/routes.rb", expected: []},
    {file: "db/migrate/20230101000000_create_users.rb", expected: []},
    {file: "README.md", expected: []},
    {file: "Gemfile", expected: []},
    {file: ".gitignore", expected: []},

    # Edge cases
    {file: "lib/my module.rb", expected: ["spec/my module_spec.rb"]},
    {file: "app/models/namespace/deeply/nested/model.rb", expected: ["spec/models/namespace/deeply/nested/model_spec.rb"]}
  ]

  mappings.each do |mapping|
    it "maps #{mapping[:file]} correctly" do
      output = run_file_mapper(mapping[:file])

      if mapping[:expected].empty?
        expect(output).to eq("#{mapping[:file]} -> (no mapping)")
      else
        expected_output = "#{mapping[:file]} -> #{mapping[:expected].join(", ")}"
        expect(output).to eq(expected_output)
      end
    end
  end

  context "controller to multiple spec types mapping (future enhancement)" do
    xit "maps controllers to controller specs AND request specs" do
      # This is a common Rails pattern where a controller has both:
      # - spec/controllers/users_controller_spec.rb (unit tests)
      # - spec/requests/users_spec.rb (integration tests)
      # - spec/requests/users_requests_spec.rb (alternative naming)

      # Current implementation only maps to controller specs
      # Future enhancement could return multiple spec files
      output = run_file_mapper("app/controllers/users_controller.rb")
      expect(output).to eq("app/controllers/users_controller.rb -> spec/controllers/users_controller_spec.rb")

      # Future implementation might return:
      # expect(output).to eq("app/controllers/users_controller.rb -> spec/controllers/users_controller_spec.rb, spec/requests/users_spec.rb")
    end

    xit "maps models to model specs AND request specs" do
      # Some teams test models via request specs too
      # Current implementation only maps to model specs
      output = run_file_mapper("app/models/user.rb")
      expect(output).to eq("app/models/user.rb -> spec/models/user_spec.rb")

      # Future implementation might check for existence and return multiple:
      # expect(output).to eq("app/models/user.rb -> spec/models/user_spec.rb, spec/requests/users_spec.rb")
    end
  end

  describe "view and template mappings (future enhancement)" do
    xit "maps view files to view specs" do
      # Views could map to view specs
      run_file_mapper("app/views/users/index.html.erb")
      # Future: expect(output).to eq("app/views/users/index.html.erb -> spec/views/users/index.html.erb_spec.rb")
    end

    xit "maps view files to controller specs" do
      # Views could also trigger their controller specs
      run_file_mapper("app/views/users/show.html.erb")
      # Future: expect(output).to eq("app/views/users/show.html.erb -> spec/controllers/users_controller_spec.rb")
    end
  end
end
