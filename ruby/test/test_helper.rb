require 'minitest/autorun'
require 'json'
require 'open3'
require 'tempfile'

# Runs a cbat program and captures stdout, stderr, and parsed VM state.
# Programs are automatically given the dump_state_on_exit header directive
# so that final VM state can be inspected.
module VMRunner
    EXEC_PATH = File.expand_path('../exec.rb', __dir__)

    Result = Struct.new(:stdout, :stderr, :state, :exit_status, keyword_init: true)

    # Run a .cbat source string. Stdin inputs are fed line-by-line.
    def self.run(cbat_source, stdin_inputs: [], timeout: 5)
        # Inject dump_state_on_exit if not already present
        unless cbat_source.include?("dump_state_on_exit")
            cbat_source = cbat_source.sub(/^\.header\b/m, ".header\n    dump_state_on_exit")
        end

        tmpfile = Tempfile.new(['cbat_test', '.cbat'])
        tmpfile.write(cbat_source)
        tmpfile.close

        stdin_data = stdin_inputs.map { |s| s.to_s + "\n" }.join

        stdout, stderr, status = Open3.capture3(
            "ruby", EXEC_PATH, tmpfile.path,
            stdin_data: stdin_data,
            chdir: File.expand_path('..', __dir__)
        )

        state = parse_state(stderr)

        Result.new(
            stdout: stdout,
            stderr: stderr,
            state: state,
            exit_status: status
        )
    ensure
        tmpfile&.unlink
    end

    # Run an existing .cbat fixture file.
    def self.run_file(path, stdin_inputs: [], timeout: 5)
        source = File.read(path)
        run(source, stdin_inputs: stdin_inputs, timeout: timeout)
    end

    def self.parse_state(stderr)
        match = stderr[/---CBAT_STATE_BEGIN---\n(.+?)\n---CBAT_STATE_END---/m, 1]
        return nil if match.nil?
        JSON.parse(match)
    rescue JSON::ParserError
        nil
    end
end
