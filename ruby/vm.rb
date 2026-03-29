require 'json'
require './util'
require './state'
require './instructions'
require './cbat'
require './debugger'

class Program
    include Debugger

    attr_accessor :file_name, :entry_point, :version
    attr_accessor :instructions

    attr_accessor :var_lt, :subr_lt, :file_lt, :label_lt
    attr_accessor :exec_ctx, :debug_log_enable
    attr_accessor :dump_state_on_exit

    attr_accessor :current_instr, :prev_instr
    attr_accessor :instructions_executed

    def initialize(var_lt, subr_lt, file_lt)
        @label_lt = LabelLookupTable.new
        @var_lt = var_lt.nil? ? VariableLookupTable.new : var_lt
        @subr_lt = subr_lt.nil? ? SubroutineLookupTable.new : subr_lt
        @file_lt = file_lt.nil? ? FileLookupTable.new : file_lt
        @instructions = []
        @exec_ctx = :idle
        @dump_state_on_exit = false
        @instructions_executed = 0
        @ra_stack = []
    end

    def run()
        @exec_ctx = :running
        @current_instr = @entry_point

        while @exec_ctx == :running

            if @instructions[@current_instr].nil?
                @exec_ctx = :eof
                next
            end

            @prev_instr = @current_instr.dup
            @var_lt.store("PC", @current_instr)

            if @debug_log_enable and @current_instr == @entry_point
                @debug_step = true
                debug
            end

            if @debug_step
                puts "[step] #{@current_instr} | #{@instructions[@current_instr].to_cbat}"
            end

            @instructions_executed += 1
            instr = @instructions[@current_instr]

            case instr
            when TerminateInstruction
                @exec_ctx = :terminated

            when LabelInstruction
                @label_lt.store(instr.args[0], @current_instr)
                @current_instr += 1

            when GotoInstruction
                @current_instr = instr.target(@current_instr)

            when GotoAddressInstruction
                @current_instr = instr.target

            when GotoSubroutineInstruction
                @ra_stack.push(@current_instr + 1)
                @var_lt.store("RA", @current_instr + 1)
                @current_instr = instr.target(@current_instr)

            when ReturnInstruction
                @current_instr = @ra_stack.pop || @var_lt.get("RA").to_i

            when BreakpointInstruction
                debug
                @current_instr += 1

            when IfEqualInstruction, IfNotEqualInstruction,
                IfEqualIntegerInstruction, IfNotEqualIntegerInstruction,
                IfGreaterOrEqualIntegerInstruction, IfLessOrEqualIntegerInstruction,
                IfGreaterThanIntegerInstruction, IfLessThanIntegerInstruction,
                IfFileExistsInstruction, IfNotFileExistsInstruction
                @current_instr = instr.target(@current_instr)
                puts "[debug] branch target #{@current_instr} (prev #{@prev_instr})" if @debug_log_enable

            when CallInstruction
                subroutine = @subr_lt.get(instr.target)

                if subroutine.nil?
                    puts "cbat: subroutine `#{instr.target}` not found"
                    @current_instr += 1
                    next
                end

                @exec_ctx = :subroutine
                subroutine.run
                @exec_ctx = :running
                @current_instr += 1

            else
                instr.exec()
                @current_instr += 1
            end
        end

        dump_state if @dump_state_on_exit
    end

    def state_hash
        vars = {}
        @var_lt.lt.each { |k, v| vars[k.to_s] = v.to_s }
        labels = {}
        @label_lt.lt.each { |k, v| labels[k.to_s] = v.to_i }
        files = {}
        @file_lt.lt.each { |k, v| files[k.to_s] = v.to_s }
        {
            "exit_reason" => @exec_ctx.to_s,
            "pc" => @current_instr,
            "instructions_executed" => @instructions_executed,
            "variables" => vars,
            "labels" => labels,
            "files" => files
        }
    end

    def dump_state
        $stderr.puts "---CBAT_STATE_BEGIN---"
        $stderr.puts JSON.generate(state_hash)
        $stderr.puts "---CBAT_STATE_END---"
    end

    def dump_cbat
        @instructions.map do |instr|
            puts instr.to_cbat
        end
        puts "(eof)"
    end

    def dump_batch 
        puts "#{@file_name}"
        @instructions.map do |instr|
            puts instr.to_batch
        end
        puts "(eof)"
    end

    def to_cbat_file
        puts ".header"
        puts "\tfilename #{@file_name}"
        puts "\tentry #{@entry_point}"
        puts "\tver #{@version}"
        puts ".labels"
        @label_lt.map do |k,v|
            puts "\t#{k}:#{v}"
        end   
        puts ".instrs"  
        @instructions.map do |instr|
            puts "\t#{instr.to_cbat}"
        end
        puts "\t"
    end
end
