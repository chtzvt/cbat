require_relative 'test_helper'

class TestVMDispatch < Minitest::Test

    # --- labels and goto ---

    def test_goto_label
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                g skip
                st X,"wrong"
                trm
                l skip
                st X,"right"
                trm
        CBAT
        assert_equal "right", r.state["variables"]["x"]
    end

    def test_goto_address
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                ga 2
                st X,"wrong"
                st X,"right"
                trm
        CBAT
        assert_equal "right", r.state["variables"]["x"]
    end

    # --- conditionals (string) ---

    def test_ieq_with_label_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st V,"yes"
                ieq V,"yes",match
                st RESULT,"not taken"
                trm
                l match
                st RESULT,"taken"
                trm
        CBAT
        assert_equal "taken", r.state["variables"]["result"]
    end

    def test_ieq_with_label_not_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st V,"no"
                ieq V,"yes",match
                st RESULT,"not taken"
                trm
                l match
                st RESULT,"taken"
                trm
        CBAT
        assert_equal "not taken", r.state["variables"]["result"]
    end

    def test_ieq_without_label_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st V,"yes"
                ieq V,"yes"
                st RESULT,"taken"
                st RESULT,"overwritten"
                trm
        CBAT
        # When taken without label: execute next, then continue
        assert_equal "overwritten", r.state["variables"]["result"]
    end

    def test_ieq_without_label_not_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st V,"no"
                ieq V,"yes"
                st RESULT,"should skip"
                st RESULT,"fell through"
                trm
        CBAT
        # When not taken without label: skip next instruction
        assert_equal "fell through", r.state["variables"]["result"]
    end

    def test_inq_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st V,"a"
                inq V,"b",match
                st RESULT,"not taken"
                trm
                l match
                st RESULT,"taken"
                trm
        CBAT
        assert_equal "taken", r.state["variables"]["result"]
    end

    # --- conditionals (integer) ---

    def test_ieiq_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,42
                ieiq N,42,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_iniq_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,10
                iniq N,20,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_igeqi_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,10
                igeqi N,10,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_ileqi_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,5
                ileqi N,10,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_igti_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,11
                igti N,10,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_igti_not_taken_equal
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,10
                igti N,10,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "no", r.state["variables"]["r"]
    end

    def test_ilti_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,5
                ilti N,10,match
                st R,"no"
                trm
                l match
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    # --- file existence conditionals ---

    def test_iex_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "F.txt","data"
            .instrs
                iex "F.txt",found
                st R,"no"
                trm
                l found
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    def test_iex_not_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                iex "NOPE.txt",found
                st R,"no"
                trm
                l found
                st R,"yes"
                trm
        CBAT
        assert_equal "no", r.state["variables"]["r"]
    end

    def test_inx_taken
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                inx "NOPE.txt",notfound
                st R,"no"
                trm
                l notfound
                st R,"yes"
                trm
        CBAT
        assert_equal "yes", r.state["variables"]["r"]
    end

    # --- jsub / ret ---

    def test_jsub_ret
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st R,"before"
                jsub mysub
                st R,"after"
                trm
                l mysub
                st INSIDE,"yes"
                ret
        CBAT
        assert_equal "after", r.state["variables"]["r"]
        assert_equal "yes", r.state["variables"]["inside"]
    end

    def test_jsub_nested
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                jsub outer
                st R,"done"
                trm
                l outer
                st A,"outer"
                jsub inner
                st B,"back in outer"
                ret
                l inner
                st C,"inner"
                ret
        CBAT
        assert_equal "done", r.state["variables"]["r"]
        assert_equal "outer", r.state["variables"]["a"]
        assert_equal "inner", r.state["variables"]["c"]
    end

    # --- call (subroutine in separate program) ---

    def test_call_subroutine
        r = VMRunner.run <<~CBAT
            .global
                entry "MAIN.BAT"
            .header
                filename "MAIN.BAT"
            .instrs
                st X,"before"
                c "SUB.BAT"
                st X,"after"
                trm
            .header
                filename "SUB.BAT"
            .instrs
                st Y,"from sub"
                trm
        CBAT
        assert_equal "after", r.state["variables"]["x"]
        assert_equal "from sub", r.state["variables"]["y"]
    end

    # --- terminate / eof ---

    def test_terminate
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,"alive"
                trm
                st X,"dead"
        CBAT
        assert_equal "alive", r.state["variables"]["x"]
        assert_equal "terminated", r.state["exit_reason"]
    end

    def test_eof
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,"alive"
        CBAT
        assert_equal "alive", r.state["variables"]["x"]
        assert_equal "eof", r.state["exit_reason"]
    end

    # --- looping ---

    def test_counting_loop
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st I,0
                l loop
                adi I,1
                ilti I,5,loop
                trm
        CBAT
        assert_equal "5", r.state["variables"]["i"]
    end
end
