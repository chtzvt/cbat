module Debugger
    attr_accessor :debug_step, :exec_ctx

    def debugger_repl
        puts ">>> Debugger enters. Hello!"
        while (input = STDIN.gets.chomp) != 'q'
            cmd = input.split(' ', 2)
            case cmd[0]
            when "pc"
                puts "Current instruction: #{@current_instr}"
            when "d"
                puts "Internal instruction dump: \n"
                ix = 0
                @instructions.map do |i|
                    puts " #{ix} #{ix == @current_instr ? "@" : "|"} #{i.inspect}" 
                    ix += 1
                end
            when "dc"
                puts "CBAT instruction dump: \n"
                ix = 0
                @instructions.map do |i|
                    puts " #{ix} #{ix == @current_instr ? "@" : "|"} #{i.to_cbat}" 
                    ix += 1
                end
            when "insi"
                puts "Instruction: "
                inp = STDIN.gets.chomp

                puts "Target address: "
                addr = STDIN.gets.chomp.to_i

                ins = inp.split(' ', 2)
                instr_map = InstructionMap.instance
                i = instr_map.lookup(ins[0])

                if i.class == IllegalInstruction
                    puts "cbat: illegal instruction `#{ins[0]}`"
                    next
                end 
        
                i.init(ins[1], @var_lt, @label_lt, @file_lt, @exec_ctx, @debug_log_enable, @subr_lt)
                insert_instr(i, addr)
                @current_instr += 1 if addr <= @current_instr
            when "j"
                puts "Destination address: "
                @current_instr = STDIN.gets.chomp.to_i
            when "deli"
                puts "Delete target address: "
                target = STDIN.gets.chomp.to_i 
                @current_instr -= 1 if target <= @current_instr
                @instructions.delete_at(target)
            when "vset"
                puts "Variable name: "
                vname = STDIN.gets.chomp

                puts "Value: "
                vval = STDIN.gets.chomp
                @var_lt.store(vname, vval)
            when "vdel"
                puts "Variable name: "
                vname = STDIN.gets.chomp
                @var_lt.delete(vname)
            when "vdmp"
                puts "Variable dump: \n"
                ix = 0
                @var_lt.map do |k,v|
                    puts " #{k} | #{v}" 
                end
            when "fdmp"
                puts "File dump: \n"
                ix = 0
                @file_lt.map do |k,v|
                    puts " #{k} | #{v}" 
                end
            when "lbc"
                puts "Label name: "
                lbname = STDIN.gets.chomp

                puts "Destination: "
                lbdest = STDIN.gets.chomp
                @label_lt.store(lbname, lbdest.to_i)
            when "lbd"
                puts "Label name: "
                lbname = STDIN.gets.chomp
                @label_lt.delete(lbname)
            when "lbdmp"
                puts "Label dump: \n"
                ix = 0
                @label_lt.map do |k,v|
                    puts " #{k} | #{v}" 
                end
            when "foutc"
                puts self.to_cbat_file
            when "foutb"
                puts self.dump_batch
            when "sdmp"
                puts JSON.pretty_generate(state_hash)
            when "step"
                @debug_step ^= true
                puts "step log #{@debug_step ? "enabled" : "disabled"}"
            when "dlog"
                @debug_log_enable ^= true
                puts "debug log #{@debug_log_enable ? "enabled" : "disabled"}"
            when "trm"
                @exec_ctx = :terminated
                break 
            when "fcp"
                puts "Source filename: "
                src = STDIN.gets.chomp

                puts "Dest filename: "
                dest = STDIN.gets.chomp

                @file_lt.copy(src, dest)
            when "fdel"
                puts "Filename: "
                fn = STDIN.gets.chomp

                @file_lt.delete(fn)
            when "fr"
                puts "Filename: "
                fn = STDIN.gets.chomp

                puts @file_lt.read(fn)
            when "fw"
                puts "Filename: "
                fn = STDIN.gets.chomp

                puts "Content: "
                content = STDIN.gets.chomp

                @file_lt.write(fn, content)
            else 
                puts "
                --- CBAT DEBUGGER ---

                PROGRAM COUNTER
                pc    | current PC value
                j     | Jump to address

                EXECUTION
                d     | dump instructions (internal repr)
                dc    | dump instructions (cbat repr)
                insi  | Insert instruction at address
                deli  | delete instruction at address

                VARIABLES
                vset  | set variable
                vdel  | delete variable
                vdmp  | dump variables 

                LABELS
                lbc   | create label
                lbd   | delete label
                lbdmp | dump labels
                
                FILES
                fcp   | file copy
                fdel  | file delete
                fr    | read file content
                fw    | write file content
                fdmp  | dump files

                MISC
                foutc | Dump program as cbat
                foutb | Dump program as batch
                trm   | Terminate execution
                step  | toggle step log
                dlog  | toggle step log
                "
            end
        @exec_ctx = :running 
        end
        puts ">>> Debugger exits. Bye!"
    end

    def debug 
        @exec_ctx = :breakpoint
        debugger_repl
        @exec_ctx = :running unless @exec_ctx == :terminated
    end

    def insert_instr(instruction, address)
        @instructions.insert(address, instruction)
    end
end