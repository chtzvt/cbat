class String
    def batch_remove_quotes
        self.delete_prefix('"').delete_suffix('"')
    end

    def batch_get_instr_args 
        raw_args = self.split(/,(?=(?:[^\"\']*\"[^\"\']*\")*[^\"\']*$)/)
        clean_args = []
        raw_args.map do |arg|
            clean_args << arg.batch_remove_quotes
        end
        clean_args
    end 

    def batch_interpolate_string var_lt
        self.gsub(/(%[^%]*%)/) do |n|
            var_lt.get(n.to_s[1, n.to_s.length - 2].to_sym)
        end
    end

    def batch_extract_vars
        matches = self.gsub(/(%[^%]*%)/).map{ Regexp.last_match }
        match_data = []
        matches.map do |match| 
            match_data << [
                    match.to_s[1, match.to_s.length - 2],
                    match.begin(0),
                    match.end(0)       
            ]
        end
        match_data
    end
end
