package vm_test

import (
	"testing"

	"BBS_VM/pkg/parser"
	"BBS_VM/pkg/store"
	"BBS_VM/pkg/vm"
)

// BF "H" printer: 256-cell init + 9 loop iterations of BF execution
var bfHello = `.header
    filename "bench.bat"
.files
    "PROGRAM","+++++++++[>++++++++<-]>.&"
    "INPUT",""
.instrs
    st INS_CHR,""
    st INP_PTR,0
    st INS_PTR,0
    st DTA_PTR,0
    st DTA_VAL,0
    st STACK_SP,0
    st LOOP_NEST,0
    l init_data
        wf "0","DATA/%DTA_VAL%"
        adi DTA_VAL,1
        ilti DTA_VAL,256,init_data
    st DTA_VAL,0
    l load
        stfi INS_CHR,"PROGRAM","%INS_PTR%"
        ieq INS_CHR,"&"
        jsub bf_end
        ieq INS_CHR,"+"
        jsub incr_val
        ieq INS_CHR,"-"
        jsub decr_val
        ieq INS_CHR,">"
        jsub incr_ptr
        ieq INS_CHR,"<"
        jsub decr_ptr
        ieq INS_CHR,"["
        jsub loop_begin
        ieq INS_CHR,"]"
        jsub loop_end
        ieq INS_CHR,","
        jsub read_val
        ieq INS_CHR,"."
        jsub print_val
        adi INS_PTR,1
        g load
    l incr_val
        stf DTA_VAL,"DATA/%DTA_PTR%"
        adi DTA_VAL,1
        wf "%DTA_VAL%","DATA/%DTA_PTR%"
        ret
    l decr_val
        stf DTA_VAL,"DATA/%DTA_PTR%"
        sbi DTA_VAL,1
        wf "%DTA_VAL%","DATA/%DTA_PTR%"
        ret
    l incr_ptr
        adi DTA_PTR,1
        ret
    l decr_ptr
        sbi DTA_PTR,1
        ret
    l print_val
        stf DTA_VAL,"DATA/%DTA_PTR%"
        itoa TMP,"%DTA_VAL%"
        e "%TMP%"
        ret
    l read_val
        stfi INS_CHR,"INPUT","%INP_PTR%"
        ieq INS_CHR,""
        j skip_read
        atoi DTA_VAL,"%INS_CHR%"
        wf "%DTA_VAL%","DATA/%DTA_PTR%"
        adi INP_PTR,1
    l skip_read
        ret
    l loop_begin
        stf DTA_VAL,"DATA/%DTA_PTR%"
        ieiq DTA_VAL,0,skip_loop
        adi STACK_SP,1
        wf "%INS_PTR%","LOOP/%STACK_SP%"
        ret
    l skip_loop
        st LOOP_NEST,1
    l loop_scan
        adi INS_PTR,1
        stfi INS_CHR,"PROGRAM","%INS_PTR%"
        ieq INS_CHR,"[",inc_nest
        ieq INS_CHR,"]",dec_nest
        g loop_scan
    l inc_nest
        adi LOOP_NEST,1
        g loop_scan
    l dec_nest
        sbi LOOP_NEST,1
        ieiq LOOP_NEST,0,loop_done
        g loop_scan
    l loop_done
        ret
    l loop_end
        stf DTA_VAL,"DATA/%DTA_PTR%"
        ieiq DTA_VAL,0,exit_loop
        stf INS_PTR,"LOOP/%STACK_SP%"
        ret
    l exit_loop
        sbi STACK_SP,1
        ret
    l bf_end
        trm`

// Pure counting loop — measures raw dispatch overhead
var countLoop = `.header
    filename "bench.bat"
.instrs
    st I,0
    l loop
    adi I,1
    ilti I,100000,loop
    trm`

// Math-heavy: prime factorization
var primeSource = `.header
    filename "prime.bat"
.instrs
    st PRIME,600851475143
    st I,2
    l while
    ileqi I,"%PRIME%",loop
    g done
    l loop
    mod RESULT,PRIME,I
    ieiq RESULT,0,divide
    g incr
    l divide
    div PRIME,PRIME,I
    sbi I,1
    l incr
    adi I,1
    g while
    l done
    e "%I%"
    trm`

func runBench(b *testing.B, source string) {
	res, err := parser.ParseString(source)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := store.NewMemoryStore()
		io := vm.NewStringIO(nil)
		v := vm.New(s, io)
		v.LoadResult(res.GlobalEntry, res.Programs, res.Files)
		v.Run()
	}
}

func BenchmarkCountLoop(b *testing.B)  { runBench(b, countLoop) }
func BenchmarkPrime(b *testing.B)      { runBench(b, primeSource) }
func BenchmarkBFHello(b *testing.B)    { runBench(b, bfHello) }
