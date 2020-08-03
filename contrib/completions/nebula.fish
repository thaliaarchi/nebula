function __fish_nebula_first_arg -d 'Returns true for the first argument'
  set -l tokens (commandline -opc)
  not count $tokens[2..-1]
end

function __fish_nebula_command -a command description
  complete -c nebula -f -a $command -n '__fish_nebula_first_arg' -d $description
  # TODO register -h -help --help
end

function __fish_nebula_comand_flag -a command flag description
  complete -c nebula -l $flag -o $flag -d $description
end

function __fish_nebula_command_arg -a command arg description

end

__fish_nebula_command pack 'Compress program to bit packed format'
__fish_nebula_command unpack 'Uncompress program from bit packed format'
__fish_nebula_command graph 'Print Nebula IR control flow graph'
__fish_nebula_command ast 'Emit Whitespace AST'
__fish_nebula_command ir 'Emit Nebula IR'
__fish_nebula_command llvm 'Emit LLVM IR'
__fish_nebula_command help 'Print usage'

__fish_nebula_command_flag graph ascii 'Print as ASCII grid rather than DOT digraph'
__fish_nebula_command_flag graph nofold 'Disable constant folding'
__fish_nebula_command_flag_option ast format 'Output format' ws wsa wsapos wsx
__fish_nebula_command_flag ir nofold 'Disable constant folding'
__fish_nebula_command_flag_uint llvm calls 'Maximum call stack length for LLVM codegen'
__fish_nebula_command_flag_uint llvm heap 'Maximum heap address bound for LLVM codegen'
__fish_nebula_command_flag llvm nofold 'Disable constant folding'
__fish_nebula_command_flag_uint llvm stack 'Maximum stack length for LLVM codegen'

complete -c nebula -l help -o help -s h --description 'Print usage'

for help in -h -help --help help
  __fish_nebula_command_arg $help pack 'Compress program to bit packed format'
  __fish_nebula_command_arg $help unpack 'Uncompress program from bit packed format'
  __fish_nebula_command_arg $help graph 'Print Nebula IR control flow graph'
  __fish_nebula_command_arg $help ast 'Emit Whitespace AST'
  __fish_nebula_command_arg $help ir 'Emit Nebula IR'
  __fish_nebula_command_arg $help llvm 'Emit LLVM IR'
end
