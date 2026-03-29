require 'singleton'

class InstructionMap 
    include Singleton 

    def lookup(op)
        case op
        when "l"
            LabelInstruction.new
        when "e"
            EchoInstruction.new
        when "stp"
            SetPromptInstruction.new
        when "af"
            AppendFileInstruction.new
        when "wf"
            WriteFileInstruction.new
        when "wfi"
            WriteFileIndexInstruction.new
        when "trm"
            TerminateInstruction.new
        when "st"
            StoreInstruction.new
        when "stfi"
            StoreFromFileIndexInstruction.new
        when "stf"
            StoreFromFileInstruction.new
        when "bp"
            BreakpointInstruction.new
        when "nop"
            NopInstruction.new
        when "g"
            GotoInstruction.new
        when "j"
            GotoInstruction.new
        when "ga"
            GotoAddressInstruction.new
        when "ja"
            GotoAddressInstruction.new
        when "gsub"
            GotoSubroutineInstruction.new
        when "jsub"
            GotoSubroutineInstruction.new
        when "ret"
            ReturnInstruction.new
        when "p"
            PauseInstruction.new
        when "ieq"
            IfEqualInstruction.new
        when "inq"
            IfNotEqualInstruction.new
        when "igeqi"
            IfGreaterOrEqualIntegerInstruction.new
        when "ileqi"
            IfLessOrEqualIntegerInstruction.new
        when "ieiq"
            IfEqualIntegerInstruction.new
        when "iniq"
            IfNotEqualIntegerInstruction.new
        when "iex"
            IfFileExistsInstruction.new
        when "inx"
            IfNotFileExistsInstruction.new
        when "adi"
            AddImmediateInstruction.new
        when "sbi"
            SubtractImmediateInstruction.new
        when "mli"
            MultiplyImmediateInstruction.new
        when "dvi"
            DivideImmediateInstruction.new
        when "mdi"
            ModuloImmediateInstruction.new
        when "add"
            AddInstruction.new
        when "sub"
            SubtractInstruction.new
        when "mul"
            MultiplyInstruction.new
        when "div"
            DivideInstruction.new
        when "mod"
            ModuloInstruction.new
        when "cls"
            ClearScreenInstruction.new
        when "clr"
            ColorInstruction.new
        when "t"
            TypeFileInstruction.new
        when "c"
            CallInstruction.new
        when "atoi"
            AsciiToInteger.new
        when "itoa"
            IntegerToAscii.new
        when "df"
            DeleteFileInstruction.new
        when "mkd"
            MakeDirectoryInstruction.new
        when "slp"
            SleepInstruction.new
        when "ttl"
            TitleInstruction.new
        when "igti"
            IfGreaterThanIntegerInstruction.new
        when "ilti"
            IfLessThanIntegerInstruction.new
        when "cat"
            ConcatenateInstruction.new
        when "len"
            LengthInstruction.new
        when "sta"
            StoreArithmeticInstruction.new
        when "tp"
            TypePagedInstruction.new
        else
            NopInstruction.new
        end
    end
end

module Executable
    attr_accessor :args, :raw_args, :var_lt, :label_lt, :file_lt, :subr_lt, :ec, :debug_enable

    def init(raw_str, var_lt, label_lt, file_lt, ec, debug_enable, subr_lt = nil)
        @args = raw_str.batch_get_instr_args unless raw_str.nil?
        @raw_args = @args.clone.map(&:clone) unless raw_str.nil?
        @var_lt = var_lt
        @label_lt = label_lt
        @file_lt = file_lt
        @subr_lt = subr_lt
        @ec = ec
        @debug_enable = debug_enable
    end

    def exec
        raise "Not implemented!"
    end

    def to_batch
        raise "Not implemented!"
    end

    def to_cbat
        raise "Not implemented!"
    end

    def op
        to_cbat.split(' ')[0]
    end

    def run(args, var_lt, label_lt, file_lt, pc)
        self.init(args, var_lt, label_lt, file_lt, pc)
        self.exec()
    end
end

class EchoInstruction
    include Executable

    def exec
        puts @args[0].batch_interpolate_string(@var_lt)
    end

    def to_batch
        "echo \"#{@raw_args[0]}\""
    end

    def to_cbat
        "e \"#{@raw_args[0]}\""
    end
end

class ClearScreenInstruction
    include Executable

    def exec
        puts "\e[H\e[2J" unless @debug_enable
    end

    def to_batch
        "cls"
    end

    def to_cbat
        "cls"
    end
end

class StoreInstruction
    include Executable

    def exec
        puts "[debug] store #{@args[0]}='#{@args[1]}'" if @debug_enable
        @var_lt.store(@args[0], @args[1].batch_interpolate_string(@var_lt))
    end

    def to_batch
        "set #{@raw_args[0]}=\"#{@raw_args[1]}\""
    end

    def to_cbat
        "st #{@raw_args[0]},\"#{@raw_args[1]}\""
    end
end

class StoreFromFileInstruction
    include Executable

    def exec
        puts "[debug] store #{@args[0]} from file '#{@args[1]}'" if @debug_enable
        @var_lt.store(@args[0], @file_lt.read(@args[1].batch_interpolate_string(@var_lt)))
    end

    def to_batch
        "::load from file #{@raw_args[0]}=\"#{@raw_args[1]}\""
    end

    def to_cbat
        "stf #{@raw_args[0]},\"#{@raw_args[1]}\""
    end
end

class StoreFromFileIndexInstruction
    include Executable

    def exec
        puts "[debug] store to #{@args[0]} from index #{@args[2]} in file '#{@args[1]}'" if @debug_enable
        @var_lt.store(@args[0], @file_lt.read(@args[1].batch_interpolate_string(@var_lt))[@args[2].batch_interpolate_string(@var_lt).to_i])
    end

    def to_batch
        "::load from file index #{@raw_args[0]}=\"#{@raw_args[1]}\""
    end

    def to_cbat
        "stfi #{@raw_args[0]},\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class SetPromptInstruction
    include Executable

    def prompt(msg)
        print(msg.batch_remove_quotes)
        STDIN.gets
    end

    def exec
        str = args[1]
        str = "Input: " if args[1].empty?
        puts "[debug] prompt store #{@args[0]}" if @debug_enable
        @var_lt.store(args[0].to_sym, prompt(str).chomp)
    end

    def to_batch
        "set /p #{@raw_args[0]}=\"#{@raw_args[1]}\""
    end

    def to_cbat
        "stp #{@raw_args[0]},\"#{@raw_args[1]}\""
    end
end

class PauseInstruction
    include Executable

    def prompt(msg)
        puts msg.batch_remove_quotes unless msg.empty?
        STDIN.gets
    end

    def exec
        if args.nil? or args[0].to_i != 0
            str = "Press any key to continue..."
        else 
            str = ""
        end 

        prompt(str).chomp
    end

    def to_batch
        if args.nil? or args[0].to_i != 0
            "pause"
        else 
            "pause>nul"
        end 
    end

    def to_cbat
        if args.nil? or args[0].to_i != 0
            "p"
        else 
            "p 0"
        end 
    end
end

class AppendFileInstruction
    include Executable

    def exec
        puts "[debug] file append #{@args[1]} '#{@args[0].batch_interpolate_string(@var_lt)}'" if @debug_enable
        @file_lt.append(@args[1].batch_interpolate_string(@var_lt), @args[0].batch_interpolate_string(@var_lt))
    end

    def to_batch
        "echo \"#{@raw_args[0]}\" >>\"#{@raw_args[1]}\""
    end

    def to_cbat
        "af \"#{@raw_args[0]}\",\"#{@raw_args[1]}\""
    end
end

class WriteFileInstruction
    include Executable

    def exec
        puts "[debug] file write/overwrite #{@args[1]} '#{@args[0].batch_interpolate_string(@var_lt)}'" if @debug_enable
        @file_lt.write(@args[1].batch_interpolate_string(@var_lt), @args[0].batch_interpolate_string(@var_lt))
    end

    def to_batch
        "echo \"#{@raw_args[0]}\" >\"#{@raw_args[1]}\""
    end

    def to_cbat
        "wf \"#{@raw_args[0]}\",\"#{@raw_args[1]}\""
    end
end

class WriteFileIndexInstruction
    include Executable

    def exec
        puts "[debug] write value #{@args[2]} at index #{@args[1]} in file '#{@args[0]}'" if @debug_enable
        filename = @args[0].batch_interpolate_string(@var_lt)
        index = @args[1].batch_interpolate_string(@var_lt).to_i
        original_file_content = @file_lt.read(filename)
        original_file_content[index] = @args[2].batch_interpolate_string(@var_lt)
        @file_lt.write(@args[0].batch_interpolate_string(@var_lt), original_file_content)
    end

    def to_batch
        "::load from file index #{@raw_args[0]}=\"#{@raw_args[1]}\""
    end

    def to_cbat
        "wfi #{@raw_args[0]},\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class TypeFileInstruction
    include Executable

    def exec
        puts "[debug] file type '#{@args[0].batch_interpolate_string(@var_lt)}'" if @debug_enable
        print @file_lt.read(@args[0].batch_interpolate_string(@var_lt))
    end

    def to_batch
        "type \"#{@raw_args[0]}\""
    end

    def to_cbat
        "t \"#{@raw_args[0]}\""
    end
end

class IfEqualInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if equal compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]) == @args[1].batch_interpolate_string(@var_lt)
            if @args[2].nil?
                puts "[debug] \tif equal comparison succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif equal comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i})" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else 
            if @args[2].nil?
                puts "[debug] \tif equal comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif equal comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if \"%#{@raw_args[0]}%\" EQU \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "ieq \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class IfNotEqualInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if not equal compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]) != @args[1].batch_interpolate_string(@var_lt)
            if @args[2].nil?
                puts "[debug] \tif not equal comparison succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif not equal comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i}" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else 
            if @args[2].nil?
                puts "[debug] \tif not equal comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif not equal comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if \"%#{@raw_args[0]}%\" NOT EQU \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "inq \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class IfEqualIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if equal integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i == @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                puts "[debug] \tif equal integer comparison succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif equal integer comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i})" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else 
            if @args[2].nil?
                puts "[debug] \tif equal integer comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif equal integer comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if \"%#{@raw_args[0]}%\" EQU \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "ieiq \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class IfNotEqualIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if not equal integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i != @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                puts "[debug] \tif not equal integer comparison succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif not equal integer comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i}" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else 
            if @args[2].nil?
                puts "[debug] \tif not equal integer comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif not equal integer comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if \"%#{@raw_args[0]}%\" NOT EQU \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "iniq \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class IfGreaterOrEqualIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if greater or equal integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i >= @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                puts "[debug] \tif greater or equal integer comparison succeeds, advancing" if @debug_enable
                ci + 1
            else
                puts "[debug] \tif greater or equal integer comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i}" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else
            if @args[2].nil?
                puts "[debug] \tif greater or equal integer comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else
                puts "[debug] \tif greater or equal integer comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end

    def to_batch
        "if \"%#{@raw_args[0]}%\" GEQ \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "igeqi \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end

class IfLessOrEqualIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if less or equal integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i <= @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                puts "[debug] \tif less or equal integer comparison succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif less or equal integer comparison succeeds, jumping to #{@args[2]} (#{@label_lt.get(@args[2]).to_i}" if @debug_enable
                @label_lt.get(@args[2]).to_i
            end
        else 
            if @args[2].nil?
                puts "[debug] \tif less or equal integer comparison failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif less or equal integer comparison failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if \"%#{@raw_args[0]}%\" LEQ \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "ileqi \"#{@raw_args[0]}\",\"#{@raw_args[1]}\",#{@raw_args[2]}"
    end
end


class IfFileExistsInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if file exists '#{args[0].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @file_lt.file_exists?(args[0].batch_interpolate_string(@var_lt))
            if @args[1].nil?
                puts "[debug] \tif file exists succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif file exists succeeds, jumping to #{@args[1]} (#{@label_lt.get(@args[1]).to_i}" if @debug_enable
                @label_lt.get(@args[1]).to_i
            end
        else 
            if @args[1].nil?
                puts "[debug] \tif file exists failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif file exists failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if EXIST \"%#{@raw_args[0]}%\" goto #{@raw_args[1]}"
    end

    def to_cbat
        "iex \"#{@raw_args[0]}\",#{@raw_args[1]}"
    end
end

class IfNotFileExistsInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if file not exists '#{args[0].batch_interpolate_string(@var_lt)}'" if @debug_enable
        unless @file_lt.file_exists?(args[0].batch_interpolate_string(@var_lt))
            if @args[1].nil?
                puts "[debug] \tif file not exists succeeds, advancing" if @debug_enable
                ci + 1
            else 
                puts "[debug] \tif file not exists succeeds, jumping to #{@args[1]} (#{@label_lt.get(@args[1]).to_i}" if @debug_enable
                @label_lt.get(@args[1]).to_i
            end
        else 
            if @args[1].nil?
                puts "[debug] \tif file not exists failed without a label, advancing" if @debug_enable
                ci + 2
            else 
                puts "[debug] \tif file not exists failed with label, advancing" if @debug_enable
                ci + 1
            end
        end
    end 

    def to_batch
        "if NOT EXIST \"%#{@raw_args[0]}%\" goto #{@raw_args[1]}"
    end

    def to_cbat
        "inx \"#{@raw_args[0]}\",#{@raw_args[1]}"
    end
end

class GotoInstruction
    include Executable

    def exec
        @ec = :jump
        target
    end

    def target(cur = nil)
        case @args[0].batch_interpolate_string(@var_lt).downcase.to_sym
        when :cbat_next
            (cur || 0) + 1
        when :cbat_prev
            (cur || 0) - 1
        else
            puts "[debug] goto target #{@args[0].batch_interpolate_string(@var_lt)}@#{@label_lt.get(@args[0].batch_interpolate_string(@var_lt)).to_i}" if @debug_enable
            @label_lt.get(@args[0].batch_interpolate_string(@var_lt)).to_i
        end
    end

    def to_batch
        "goto #{@raw_args[0]}"
    end

    def to_cbat
        "g #{@raw_args[0]}"
    end
end

class GotoSubroutineInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(cur)
        puts "[debug] goto subroutine target #{@args[0].batch_interpolate_string(@var_lt)}@#{@label_lt.get(@args[0].batch_interpolate_string(@var_lt)).to_i}" if @debug_enable
        @var_lt.store("RA", cur + 1)
        @label_lt.get(@args[0].batch_interpolate_string(@var_lt)).to_i
    end 

    def to_batch
        "::goto subroutine #{@raw_args[0]}"
    end

    def to_cbat
        "gsub #{@raw_args[0]}"
    end
end

class GotoAddressInstruction
    include Executable

    def exec
        @ec = :jump_address
        target
    end

    def target
        puts "[debug] goto address target #{@args[0].batch_interpolate_string(@var_lt).to_i}" if @debug_enable
        @args[0].batch_interpolate_string(@var_lt).to_i
    end 

    def to_batch
        "::jump to address #{@raw_args[0]}"
    end

    def to_cbat
        "ga #{@raw_args[0]}"
    end
end

class ReturnInstruction
    include Executable

    def exec
        @ec = :jump_address
        target
    end

    def target
        ra = @var_lt.get("RA")
        puts "[debug] return to address #{ra}" if @debug_enable
        ra.to_i
    end

    def to_batch
        "::jump to address #{@raw_args[0]}"
    end

    def to_cbat
        "ret"
    end
end

class CallInstruction
    include Executable

    def exec
        @ec = :subroutine
        target
    end

    def target
        puts "[debug] call target #{@args[0]}" if @debug_enable
        @args[0].batch_interpolate_string(@var_lt)
    end

    def to_batch
        "call #{@raw_args[0]}"
    end

    def to_cbat
        "c #{@raw_args[0]}"
    end
end

class TerminateInstruction
    include Executable

    def exec
        puts "[debug] terminated" if @debug_enable
        @ec = :terminated
    end

    def to_batch
        "exit"
    end

    def to_cbat
        "trm"
    end
end

class BreakpointInstruction
    include Executable

    def exec
        @ec = :breakpoint
    end

    def to_cbat
        "bp"
    end

    def to_batch
        "::cbat:breakpoint"
    end
end

class LabelInstruction
    include Executable

    def exec
        puts "[debug] label def: #{@args[0]}" if @debug_enable
        ""
    end

    def to_cbat
        "l #{@raw_args[0]}"
    end

    def to_batch
        ":#{@raw_args[0]}"
    end
end   

class NopInstruction
    include Executable

    def exec
        puts "[debug] no-op" if @debug_enable
        ""
    end

    def to_cbat
        "nop"
    end

    def to_batch
        "::cbat:nop"
    end
end  

class IllegalInstruction
    include Executable

    def exec
        puts "[debug] illegal instr" if @debug_enable
        ""
    end

    def to_cbat
        "nop"
    end

    def to_batch
        "::cbat:nop"
    end
end   

class AddImmediateInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        value = @args[1].batch_interpolate_string(@var_lt).to_i
        puts "[debug] add immediate #{ident} #{value}" if @debug_enable
        @var_lt.store(ident, @var_lt.get(ident).to_i + value)
    end

    def to_batch
        "::add #{@raw_args[1]} to #{@raw_args[0]}"
    end

    def to_cbat
        "adi #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class SubtractImmediateInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        value = @args[1].batch_interpolate_string(@var_lt).to_i
        puts "[debug] subtract immediate #{ident} #{value}" if @debug_enable
        @var_lt.store(ident, @var_lt.get(ident).to_i - value)
    end

    def to_batch
        "::subtract #{@raw_args[1]} from #{@raw_args[0]}"
    end

    def to_cbat
        "sbi #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class MultiplyImmediateInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        value = @args[1].batch_interpolate_string(@var_lt).to_i
        puts "[debug] multiply immediate #{ident} #{value}" if @debug_enable
        @var_lt.store(ident, @var_lt.get(ident).to_i * value)
    end

    def to_batch
        "::multiply #{@raw_args[1]} by #{@raw_args[0]}"
    end

    def to_cbat
        "mli #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class DivideImmediateInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        value = @args[1].batch_interpolate_string(@var_lt).to_i
        puts "[debug] divide immediate #{ident} #{value}" if @debug_enable

        if value == 0
            puts "[debug] divide by zero!" if @debug_enable
            @var_lt.store(ident, 0)
        else 
            @var_lt.store(ident, (@var_lt.get(ident).to_i / value).to_i)
        end
    end

    def to_batch
        "::divide #{@raw_args[1]} by #{@raw_args[0]}"
    end

    def to_cbat
        "dvi #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class ModuloImmediateInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        value = @args[1].batch_interpolate_string(@var_lt).to_i
        puts "[debug] modulo immediate #{ident} #{value}" if @debug_enable

        @var_lt.store(ident, (@var_lt.get(ident).to_i % value).to_i)
    end

    def to_batch
        "::mod #{@raw_args[1]} by #{@raw_args[0]}"
    end

    def to_cbat
        "mdi #{@raw_args[0]},#{@raw_args[1]}"
    end
end


class AddInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @var_lt.get(@args[1].batch_interpolate_string(@var_lt)).to_i
        op2 = @var_lt.get(@args[2].batch_interpolate_string(@var_lt)).to_i
        puts "[debug] add #{ident} #{op1} #{op2}" if @debug_enable
        @var_lt.store(ident, op1 + op2)
    end

    def to_batch
        "set /a #{@raw_args[0]}=%#{@raw_args[1]}%+%#{@raw_args[2]}%"
    end

    def to_cbat
        "add #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class SubtractInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @var_lt.get(@args[1].batch_interpolate_string(@var_lt)).to_i
        op2 = @var_lt.get(@args[2].batch_interpolate_string(@var_lt)).to_i
        puts "[debug] subtract #{ident} #{op1} #{op2}" if @debug_enable
        @var_lt.store(ident, op1 - op2)
    end

    def to_batch
        "set /a #{@raw_args[0]}=%#{@raw_args[1]}%-%#{@raw_args[2]}%"
    end

    def to_cbat
        "sub #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class MultiplyInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @var_lt.get(@args[1].batch_interpolate_string(@var_lt)).to_i
        op2 = @var_lt.get(@args[2].batch_interpolate_string(@var_lt)).to_i
        puts "[debug] multiply #{ident} #{op1} #{op2}" if @debug_enable
        @var_lt.store(ident, op1 * op2)
    end

    def to_batch
        "set /a #{@raw_args[0]}=%#{@raw_args[1]}%*%#{@raw_args[2]}%"
    end

    def to_cbat
        "mul #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class DivideInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @var_lt.get(@args[1].batch_interpolate_string(@var_lt)).to_i
        op2 = @var_lt.get(@args[2].batch_interpolate_string(@var_lt)).to_i
        puts "[debug] divide #{ident} #{op1} #{op2}" if @debug_enable

        if op2 == 0
            puts "[debug] divide by zero!" if @debug_enable
            @var_lt.store(ident, 0)
        else
            @var_lt.store(ident, (op1 / op2).to_i)
        end
    end

    def to_batch
        "set /a #{@raw_args[0]}=%#{@raw_args[1]}%/%#{@raw_args[2]}%"
    end

    def to_cbat
        "div #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class ModuloInstruction
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @var_lt.get(@args[1].batch_interpolate_string(@var_lt)).to_i
        op2 = @var_lt.get(@args[2].batch_interpolate_string(@var_lt)).to_i
        puts "[debug] modulo #{ident} #{op1} #{op2}" if @debug_enable

        @var_lt.store(ident, (op1 % op2).to_i)
    end

    def to_batch
        "set /a #{@raw_args[0]}=%#{@raw_args[1]}%%%%#{@raw_args[2]}%"
    end

    def to_cbat
        "mod #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class AsciiToInteger
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @args[1].batch_interpolate_string(@var_lt)
        puts "[debug] atoi #{ident} #{op1}" if @debug_enable

        @var_lt.store(ident, op1.ord)
    end

    def to_batch
        "::atoi #{@raw_args[1]} by #{@raw_args[0]}"
    end

    def to_cbat
        "atoi #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class IntegerToAscii
    include Executable

    def exec
        ident = @args[0].batch_interpolate_string(@var_lt)
        op1 = @args[1].batch_interpolate_string(@var_lt)
        puts "[debug] itoa #{ident} #{op1}" if @debug_enable

        @var_lt.store(ident, op1.to_i.chr)
    end

    def to_batch
        "::itoa #{@raw_args[1]} by #{@raw_args[0]}"
    end

    def to_cbat
        "itoa #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class DeleteFileInstruction
    include Executable

    def exec
        path = @args[0].batch_interpolate_string(@var_lt)
        puts "[debug] delete file '#{path}'" if @debug_enable
        @file_lt.delete(path)
    end

    def to_batch
        "del \"#{@raw_args[0]}\""
    end

    def to_cbat
        "df \"#{@raw_args[0]}\""
    end
end

class MakeDirectoryInstruction
    include Executable

    def exec
        path = @args[0].batch_interpolate_string(@var_lt)
        puts "[debug] mkdir '#{path}'" if @debug_enable
        @file_lt.create(path)
    end

    def to_batch
        "mkdir \"#{@raw_args[0]}\""
    end

    def to_cbat
        "mkd \"#{@raw_args[0]}\""
    end
end

class SleepInstruction
    include Executable

    def exec
        ms = @args[0].batch_interpolate_string(@var_lt).to_i
        puts "[debug] sleep #{ms}ms" if @debug_enable
        sleep(ms / 1000.0)
    end

    def to_batch
        "ping 127.0.0.1 -n 1 -w #{@raw_args[0]} >nul"
    end

    def to_cbat
        "slp #{@raw_args[0]}"
    end
end

class ColorInstruction
    include Executable

    BATCH_COLOR_MAP = {
        "0" => "0",    # black
        "1" => "4",    # blue
        "2" => "2",    # green
        "3" => "6",    # aqua/cyan
        "4" => "1",    # red
        "5" => "5",    # purple/magenta
        "6" => "3",    # yellow
        "7" => "7",    # white
        "8" => "0;1",  # gray (bright black)
        "9" => "4;1",  # light blue
        "a" => "2;1",  # light green
        "b" => "6;1",  # light aqua
        "c" => "1;1",  # light red
        "d" => "5;1",  # light purple
        "e" => "3;1",  # light yellow
        "f" => "7;1",  # bright white
    }.freeze

    def exec
        code = @args[0].batch_interpolate_string(@var_lt)
        puts "[debug] color #{code}" if @debug_enable
        if code.length == 2
            bg = BATCH_COLOR_MAP[code[0].downcase] || "0"
            fg = BATCH_COLOR_MAP[code[1].downcase] || "7"
            print "\e[4#{bg.split(';')[0]}m\e[3#{fg}m"
        elsif code.length == 1
            fg = BATCH_COLOR_MAP[code[0].downcase] || "7"
            print "\e[3#{fg}m"
        end
    end

    def to_batch
        "color #{@raw_args[0]}"
    end

    def to_cbat
        "clr #{@raw_args[0]}"
    end
end

class TitleInstruction
    include Executable

    def exec
        title = @args[0].batch_interpolate_string(@var_lt)
        puts "[debug] title '#{title}'" if @debug_enable
        print "\e]0;#{title}\a"
    end

    def to_batch
        "title \"#{@raw_args[0]}\""
    end

    def to_cbat
        "ttl \"#{@raw_args[0]}\""
    end
end

class IfGreaterThanIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if greater than integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i > @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                ci + 1
            else
                @label_lt.get(@args[2]).to_i
            end
        else
            if @args[2].nil?
                ci + 2
            else
                ci + 1
            end
        end
    end

    def to_batch
        "if \"%#{@raw_args[0]}%\" GTR \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "igti #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class IfLessThanIntegerInstruction
    include Executable

    def exec
        @ec = :jump
    end

    def target(ci)
        puts "[debug] if less than integer compare '#{@var_lt.get(@args[0])}' to '#{@args[1].batch_interpolate_string(@var_lt)}'" if @debug_enable
        if @var_lt.get(@args[0]).to_i < @args[1].batch_interpolate_string(@var_lt).to_i
            if @args[2].nil?
                ci + 1
            else
                @label_lt.get(@args[2]).to_i
            end
        else
            if @args[2].nil?
                ci + 2
            else
                ci + 1
            end
        end
    end

    def to_batch
        "if \"%#{@raw_args[0]}%\" LSS \"#{@raw_args[1]}\" goto #{@raw_args[2]}"
    end

    def to_cbat
        "ilti #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class ConcatenateInstruction
    include Executable

    def exec
        dest = @args[0].batch_interpolate_string(@var_lt)
        a = @var_lt.get(@args[1].batch_interpolate_string(@var_lt))
        b = @var_lt.get(@args[2].batch_interpolate_string(@var_lt))
        puts "[debug] cat #{dest} = '#{a}' + '#{b}'" if @debug_enable
        @var_lt.store(dest, a.to_s + b.to_s)
    end

    def to_batch
        "set #{@raw_args[0]}=%#{@raw_args[1]}%%#{@raw_args[2]}%"
    end

    def to_cbat
        "cat #{@raw_args[0]},#{@raw_args[1]},#{@raw_args[2]}"
    end
end

class LengthInstruction
    include Executable

    def exec
        dest = @args[0].batch_interpolate_string(@var_lt)
        source = @var_lt.get(@args[1].batch_interpolate_string(@var_lt))
        puts "[debug] len #{dest} = len('#{source}')" if @debug_enable
        @var_lt.store(dest, source.length)
    end

    def to_batch
        "::len #{@raw_args[0]} of #{@raw_args[1]}"
    end

    def to_cbat
        "len #{@raw_args[0]},#{@raw_args[1]}"
    end
end

class StoreArithmeticInstruction
    include Executable

    def exec
        dest = @args[0].batch_interpolate_string(@var_lt)
        expr = @args[1].batch_interpolate_string(@var_lt)
        puts "[debug] sta #{dest} = #{expr}" if @debug_enable

        # Simple expression parser for batch-style arithmetic
        # Supports: +, -, *, /, % with integer operands
        result = eval_arithmetic(expr)
        @var_lt.store(dest, result)
    end

    def to_batch
        "set /a #{@raw_args[0]}=#{@raw_args[1]}"
    end

    def to_cbat
        "sta #{@raw_args[0]},\"#{@raw_args[1]}\""
    end

    private

    def eval_arithmetic(expr)
        # Tokenize: split on operators while keeping them
        tokens = expr.scan(/(-?\d+|[+\-*\/%])/).flatten.reject(&:empty?)

        return 0 if tokens.empty?

        # Evaluate with standard precedence: * / % first, then + -
        # First pass: handle * / %
        i = 0
        while i < tokens.length
            if ["*", "/", "%"].include?(tokens[i])
                left = tokens[i - 1].to_i
                right = tokens[i + 1].to_i
                result = case tokens[i]
                         when "*" then left * right
                         when "/" then right == 0 ? 0 : left / right
                         when "%" then right == 0 ? 0 : left % right
                         end
                tokens[(i-1)..(i+1)] = [result.to_s]
                i -= 1
            end
            i += 1
        end

        # Second pass: handle + -
        result = tokens[0].to_i
        i = 1
        while i < tokens.length
            op = tokens[i]
            val = tokens[i + 1].to_i
            case op
            when "+" then result += val
            when "-" then result -= val
            end
            i += 2
        end

        result
    end
end

class TypePagedInstruction
    include Executable

    def prompt_continue
        print "-- MORE --"
        STDIN.gets
    end

    def exec
        path = @args[0].batch_interpolate_string(@var_lt)
        lines_per_page = (@args[1] || "20").batch_interpolate_string(@var_lt).to_i
        puts "[debug] type paged '#{path}' (#{lines_per_page} lines/page)" if @debug_enable

        content = @file_lt.read(path)
        lines = content.split("\n")

        lines.each_with_index do |line, idx|
            puts line
            if (idx + 1) % lines_per_page == 0 && idx + 1 < lines.length
                prompt_continue
            end
        end
    end

    def to_batch
        "type \"#{@raw_args[0]}\" | MORE"
    end

    def to_cbat
        "tp \"#{@raw_args[0]}\",#{@raw_args[1]}"
    end
end