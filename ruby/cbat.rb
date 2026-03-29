require './util'
require './state'
require './vm'
require './instructions'

class CBATLoader
    attr_accessor :current_section, :cur_prog, :global_entry
    attr_accessor :var_lt, :subr_lt, :file_lt

    def initialize()
        @var_lt = VariableLookupTable.new
        @subr_lt = SubroutineLookupTable.new
        @file_lt = FileLookupTable.new
    end

    def open(path)
        current_section = :unknown

        hit_header = false
        hit_file = false

        File.open(path).each_line do |line|
            line.chomp!
            next if line.nil?

            line.strip!

            # skip comments and empty lines
            next if line.start_with?(";") || line.empty?

            # strip comments from the end of the line
            line.sub!(/;.*/, '')
            line.strip!

            case line
            when ".global"
                current_section = :global
                next
            when ".files"
                current_section = :files
                next
            when ".header"
                if hit_header
                    store_prog()
                    hit_header = false
                end 

                hit_header = true

                @cur_prog = Program.new(@var_lt, @subr_lt, @file_lt)

                current_section = :header
                next
            when ".labels"
                current_section = :labels 
                next
            when ".instrs"
                current_section = :instructions
                next
            end

            case current_section 
            when :global
                read_global_field(line)
            when :files
                read_files_field(line)
            when :header 
                read_header_field(line)
            when :labels 
                read_labels_field(line)
            when :instructions
                read_instructions_field(line)
            else 
                puts "cbat: ignoring garbage `#{line}`"
                next
            end
        end
        store_prog()
    end

    def store_prog
        return if @cur_prog.nil?
        @cur_prog.entry_point = 0 if @cur_prog.entry_point.nil?
        @subr_lt.store(@cur_prog.file_name, @cur_prog.clone)
        @cur_prog = nil
    end

    def run 
        main = @subr_lt.entry_point(@global_entry)
        main.run
    end

    def read_global_field(field)
        kv = field.split(' ')
        case kv[0]
        when "entry"
            @global_entry = kv[1]
        end
    end 

    def read_files_field(field)
        kv = field.split('","')
        @file_lt.store(kv[0], kv[1..].join(' ').delete_prefix('"').delete_suffix('"').gsub("\\n","\n"))
    end 

    def read_header_field(field)
        kv = field.split(' ')
        case kv[0]
        when "filename"
            @cur_prog.file_name = kv[1]
        when "entry"
            @cur_prog.entry_point = kv[1].to_i
        when "ver"
            @cur_prog.version = kv[1]
        when "debug"
            @cur_prog.debug_log_enable = true
        when "dump_state_on_exit"
            @cur_prog.dump_state_on_exit = true
        end
    end 

    def read_labels_field(field)
        kv = field.split(':')
        @cur_prog.label_lt.store(kv[0], kv[1])
    end

    def read_instructions_field(field)
        kv = field.split(' ', 2)
        instr_map = InstructionMap.instance

        i = instr_map.lookup(kv[0])

        case i
        when LabelInstruction
            @cur_prog.label_lt.store(kv[1], @cur_prog.instructions.length)
        when IllegalInstruction
            raise "cbat: illegal instruction `#{kv[0]}`"
        end 

        i.init(kv[1], @cur_prog.var_lt, @cur_prog.label_lt, @cur_prog.file_lt, @cur_prog.exec_ctx, @cur_prog.debug_log_enable, @subr_lt)
        @cur_prog.instructions.append(i)
    end
end
