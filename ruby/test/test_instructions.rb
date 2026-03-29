require_relative 'test_helper'

class TestInstructions < Minitest::Test

    # --- echo ---

    def test_echo
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                e "hello world"
                trm
        CBAT
        assert_includes r.stdout, "hello world"
    end

    def test_echo_interpolation
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st NAME,"alice"
                e "hi %NAME%"
                trm
        CBAT
        assert_includes r.stdout, "hi alice"
    end

    # --- store / variables ---

    def test_store
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,42
                trm
        CBAT
        assert_equal "42", r.state["variables"]["x"]
    end

    def test_store_interpolation
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,"hello"
                st B,"%A% world"
                trm
        CBAT
        assert_equal "hello world", r.state["variables"]["b"]
    end

    # --- stf (store from file) ---

    def test_stf
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "DATA.txt","contents here"
            .instrs
                stf VAR,"DATA.txt"
                trm
        CBAT
        assert_equal "contents here", r.state["variables"]["var"]
    end

    # --- stfi (store from file index) ---

    def test_stfi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "BUF","abcdef"
            .instrs
                stfi CH,"BUF",2
                trm
        CBAT
        assert_equal "c", r.state["variables"]["ch"]
    end

    # --- stp (set prompt) ---

    def test_stp
        r = VMRunner.run(<<~CBAT, stdin_inputs: ["bob"])
            .header
                filename "test.bat"
            .instrs
                stp NAME,"Enter name: "
                trm
        CBAT
        assert_equal "bob", r.state["variables"]["name"]
    end

    # --- file operations ---

    def test_af_append_file
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                af "line1","OUT.txt"
                af "line2","OUT.txt"
                trm
        CBAT
        assert_equal "line1\nline2\n", r.state["files"]["OUT.txt"]
    end

    def test_wf_write_file
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                wf "first","OUT.txt"
                wf "second","OUT.txt"
                trm
        CBAT
        assert_equal "second", r.state["files"]["OUT.txt"]
    end

    def test_wfi_write_file_index
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "BUF","abcdef"
            .instrs
                wfi "BUF",2,"X"
                trm
        CBAT
        assert_equal "abXdef", r.state["files"]["BUF"]
    end

    def test_df_delete_file
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "TEMP.txt","data"
            .instrs
                df "TEMP.txt"
                trm
        CBAT
        refute r.state["files"].key?("TEMP.txt")
    end

    def test_t_type_file
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .files
                "MSG.txt","hello from file"
            .instrs
                t "MSG.txt"
                trm
        CBAT
        assert_includes r.stdout, "hello from file"
    end

    def test_mkd
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                mkd "MYDIR"
                trm
        CBAT
        assert r.state["files"].key?("MYDIR")
    end

    # --- immediate math ---

    def test_adi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,10
                adi X,5
                trm
        CBAT
        assert_equal "15", r.state["variables"]["x"]
    end

    def test_sbi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,10
                sbi X,3
                trm
        CBAT
        assert_equal "7", r.state["variables"]["x"]
    end

    def test_mli
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,6
                mli X,7
                trm
        CBAT
        assert_equal "42", r.state["variables"]["x"]
    end

    def test_dvi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,20
                dvi X,4
                trm
        CBAT
        assert_equal "5", r.state["variables"]["x"]
    end

    def test_dvi_by_zero
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,10
                dvi X,0
                trm
        CBAT
        assert_equal "0", r.state["variables"]["x"]
    end

    def test_mdi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st X,17
                mdi X,5
                trm
        CBAT
        assert_equal "2", r.state["variables"]["x"]
    end

    # --- register-to-register math ---

    def test_add
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,10
                st B,20
                add C,A,B
                trm
        CBAT
        assert_equal "30", r.state["variables"]["c"]
    end

    def test_sub
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,50
                st B,8
                sub C,A,B
                trm
        CBAT
        assert_equal "42", r.state["variables"]["c"]
    end

    def test_mul
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,6
                st B,7
                mul C,A,B
                trm
        CBAT
        assert_equal "42", r.state["variables"]["c"]
    end

    def test_div
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,100
                st B,4
                div C,A,B
                trm
        CBAT
        assert_equal "25", r.state["variables"]["c"]
    end

    def test_mod
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,17
                st B,5
                mod C,A,B
                trm
        CBAT
        assert_equal "2", r.state["variables"]["c"]
    end

    # --- sta (arithmetic expression) ---

    def test_sta_simple
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                sta R,"2+3"
                trm
        CBAT
        assert_equal "5", r.state["variables"]["r"]
    end

    def test_sta_precedence
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                sta R,"2+3*4"
                trm
        CBAT
        assert_equal "14", r.state["variables"]["r"]
    end

    # --- cat (concatenate) ---

    def test_cat
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st A,"foo"
                st B,"bar"
                cat C,A,B
                trm
        CBAT
        assert_equal "foobar", r.state["variables"]["c"]
    end

    # --- len (length) ---

    def test_len
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st S,"hello"
                len L,S
                trm
        CBAT
        assert_equal "5", r.state["variables"]["l"]
    end

    # --- atoi / itoa ---

    def test_atoi
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                atoi CODE,"A"
                trm
        CBAT
        assert_equal "65", r.state["variables"]["code"]
    end

    def test_itoa
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                st N,65
                itoa CH,"%N%"
                trm
        CBAT
        assert_equal "A", r.state["variables"]["ch"]
    end

    # --- slp (sleep) ---

    def test_slp
        start = Time.now
        r = VMRunner.run <<~CBAT
            .header
                filename "test.bat"
            .instrs
                slp 100
                trm
        CBAT
        elapsed = Time.now - start
        assert elapsed >= 0.08, "sleep should have waited at least ~100ms"
        assert_equal "terminated", r.state["exit_reason"]
    end
end
