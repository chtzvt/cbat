module LookupTable 
    attr_reader :lt

    def initialize
        @lt = {}
    end

    def store(name, value)
        @lt.store(name.downcase.to_sym, value)
    end

    def get(name)
        @lt.fetch(name.downcase.to_sym, "undefined")
    end

    def delete(name)
        @lt.delete(name.downcase.to_sym)
    end

    def map(&block)
        @lt.map &block
    end

    def keys 
        @lt.keys 
    end

    def length 
        @lt.length 
    end

    def to_s 
        @lt.to_s 
    end
end

class VariableLookupTable
    include LookupTable

    def get(name)
        case name.downcase.to_sym
        when :date
            Time.new.strftime("%Y-%m-%d")
        when :time
            Time.new.strftime("%H:%M:%S")
        when :random
            rand(0..32767).to_s
        when :username
            `whoami`.chomp
        else
            @lt.fetch(name.downcase.to_sym, "undefined")
        end
    end
end 

class LabelLookupTable
    include LookupTable
end 

class FileLookupTable
    include LookupTable

    def store(name, value)
        @lt.store(name.delete('"').to_sym, value)
    end

    def get(name)
        @lt.fetch(name.to_sym, "undefined")
    end

    def delete(name)
        @lt.delete(name.to_sym)
    end

    def append(name, content)        
        orig_content = ""
        orig_content = self.get(name) if self.file_exists?(name) 
        self.store(name, orig_content + content + "\n")
    end

    def file_exists?(name)
        get(name) != "undefined"
    end

    def delete(name)
        @lt.delete(name.delete('"').to_sym)
    end

    def copy(source, dest)
        src = get(source) # Explicitly choosing not to error on non-existent source
        store(dest, src.dup)
    end

    def create(name)
        store(name, "")
    end

    def read(name)
        content = get(name) 
        content != "undefined" ? content : "\n"
    end

    def overwrite(name, content)
        store(name, content)
    end

    def write(name, content)
        overwrite(name, content)
    end
end 

class SubroutineLookupTable
    include LookupTable

    def store(name, value)
        name = "MAIN" if name.empty? or name.nil?
        @lt.store(name.downcase.delete('"').to_sym, value)
    end

    def get(name)
        @lt.fetch(name.downcase.delete('"').to_sym, nil)
    end

    def default
        raise "No programs loaded!" unless @lt.any?
        @lt.first[1]
    end

    def entry_point(global_entry)
        ep = get(global_entry) unless global_entry.nil?
        ep = get("MAIN") if ep.nil?
        ep = default() if ep.nil?
        ep
    end
end 
