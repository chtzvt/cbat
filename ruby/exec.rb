require './cbat'

if ARGV[0].nil?
  puts "Usage: ruby exec.rb <filename>"
  exit
end

loader = CBATLoader.new
loader.open(ARGV[0])
loader.run