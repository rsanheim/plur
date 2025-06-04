# Backspin TODO

Remeber to run the full build after significant changes via `bin/rake` - the default task runs specs and standardrb.

### Clean up where we store the data in the spec suite! ✅

* For unit test (i.e. things that don't remove saved data) we should use "spec/backspin_data", just like any other gem
* For integration tests (i.e. things that have to remove saved records) we should use "./tmp/backspin_data"
* We should never store data in "spec/backspin" - its confusing and I don't know how it keeps creeping back in

### Rename Dubplate to Record ✅
I want to have a more straight forward name here. Lets rename away from Dubplate to Record.
Update specs as you go.

* [x] change the object name from `Dubplate` to `Record`
* [x] the location for record files can remain "backspin_data" - so by default in an rspec project it would be ./spec/backspin_data
* [x] change the top level Backspin.record to be named `Backspin.call` to avoid confusion
* [x] update docs and CLAUDE.md
* [x] commit when everything is passing

### Store `first_recorded_at` in the cassette file ✅

We need to know the first recorded at time to allow automatic re-recording of the record later on.
Add it to the Record object and the record file. Do not implement 're-record' yet.

### Store the command info in the record file ✅

i.e. store the command type (Open3.capture3) and the arguments (["echo", "hello"])

I think we already have this mostly in the Command object, its mostly a matter of serializing it back and forth.

### Introduce the ability to record `system` calls
* we want to capture other system calls like `system("echo hello")`
* make the result and serialization format the same as capture3, even though this may mean adapting the results from system to match the more detailed output from capture3
* consider splitting this out, along side the capture3 stuff, into new objects that follow the same sort of pattern
* for context, we will want to add other command types later.

### Refactor the stub mechanism

We should refactor the details of stubbing out Backspin itself into a new object
Something like this, in rough outline form - note that this just example code, and I've left out details of logic that will have to move.

```ruby
   def call
      recorder = Recorder.new() # recorder is an object to setup the stubs and record the command details of any that get executed
      recorder.record_call(:system)
      recorder.record_call(:capture3)

      yield

      recorder.commands.each { |cmd| record.add_command(cmd) }

      Result.new(commands: recorder.commands, record_path: Pathname.new(record_path))
   end

   # then Recorder has the stub logic and stores commands and their results
   class Recorder
      attr_reader :commands

      def initialize
         @commands = []
      end

   def record_call(command_type)
      case command_type
      when :system
         setup_system_call_stub
      when :capture3
         setup_capture3_call_stub
      end
   end

      def setup_capture3_call_stub
         allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
            # Execute the real command
            stdout, stderr, status = original_method.call(*args)

            # Create command with interaction data
            command = Command.new(
               method_class: Open3::Capture3,
               args: cmd_args,
               stdout: stdout,
               stderr: stderr,
               status: status.exitstatus,
               recorded_at: Time.now.iso8601
            )
            @commands << command
         end
      end

      def setup_system_call_stub
         allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
         # etc
      end
```


