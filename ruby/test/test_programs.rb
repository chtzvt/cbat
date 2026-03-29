require_relative 'test_helper'

class TestPrograms < Minitest::Test

    TESTS_DIR = File.expand_path('../tests', __dir__)

    def test_prime_factorization
        r = VMRunner.run_file(File.join(TESTS_DIR, 'prime.cbat'))
        assert_includes r.stdout.strip, "6857"
    end

    def test_jsub_ret_program
        r = VMRunner.run_file(File.join(TESTS_DIR, 'test-jsub-ret.cbat'))
        lines = r.stdout.strip.split("\n")
        assert_equal "before jsub", lines[0]
        assert_equal "inside subroutine", lines[1]
        assert_equal "after jsub", lines[2]
        assert_equal "all done", lines[3]
    end

    def test_iex_inx_program
        r = VMRunner.run_file(File.join(TESTS_DIR, 'test-iex.cbat'))
        assert_includes r.stdout, "PASS: file exists"
        assert_includes r.stdout, "PASS: file does not exist"
        refute_includes r.stdout, "FAIL"
    end

    def test_new_instructions_program
        r = VMRunner.run_file(File.join(TESTS_DIR, 'test-new-instrs.cbat'))
        assert_includes r.stdout, "All new instructions passed!"
        refute_includes r.stdout, "FAIL"
    end

    def test_stf_program
        r = VMRunner.run_file(File.join(TESTS_DIR, 'test-stf-df.cbat'))
        assert_includes r.stdout, "stf loaded: hello world"
        refute_includes r.stdout, "FAIL"
    end

    def test_return_address_program
        # Needs debugger input 'q' to exit the breakpoint at the end
        r = VMRunner.run_file(File.join(TESTS_DIR, 'return-address.cbat'), stdin_inputs: ["q"])
        assert_includes r.stdout, "First print is at"
        assert_includes r.stdout, "Subroutine A prints at"
        assert_includes r.stdout, "Second print is at"
        assert_includes r.stdout, "Execution resumes from jsub at"
    end
end
