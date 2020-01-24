# Nebula

Nebula is an optimizing compiler for the Whitespace programming
language.

This project is evolving very rapidly.

## Usage

Building Nebula requires Go and the LLVM source. Graphviz can be
optionally used to render generated DOT graphs.

```sh
go get https://github.com/andrewarchi/nebula
cd nebula
go get ./...
go build
./nebula programs/interpret.out.ws ir
./nebula programs/interpret.out.ws dot | dot -Tpng > graph.png
```

### Compiling using Nebula

```sh
mkdir build
cd build
../nebula ../programs/collatz.out.ws llvm 2> collatz.ll
clang -S -emit-llvm ../codegen/lib/io.c
llvm-link -o collatz.o collatz.ll io.ll
llc collatz.o
clang -o collatz collatz.o.s
```

## Project Goals

### Compilation

- Nebula IR
  - Operations
    - Arithmetic - add, sub, mul, div, mod
    - Stack - push registers, pop length
    - Heap - load, store
    - I/O - read, print
    - Control flow - jmp, conditional jmp, call, ret
    - Assertions - zero divisor, stack underflow, ret underflow
  - Memory
    - Stack
    - Heap - partitioned
    - Call stack
  - Static single assignment form
    - [x] Stack access converted to indexed variables
    - [x] Connected graph of basic blocks
    - [x] Minimize stack pointer updates to once per block
  - Interprocedural analysis
    - [x] Constant stack folding and propagation
    - [x] String concatenation
    - [x] Dead code elimination
    - [x] Call stack underflow detection
    - [ ] Trap eliding
    - [ ] Register allocation
    - [ ] Reduction of arbitrary precision integers to fixed width
    - [ ] Tail call optimization
    - [ ] Heap access analysis
      - [ ] Lifetime analysis
      - [ ] Constant folding and propagation
    - [ ] Contract assertions
      - [ ] Using an SMT solver
      - [ ] Abstract interpretation
  - Targets
    - [x] LLVM IR
    - [ ] WebAssembly via Emscripten

### Languages

- High level language
  - Abstracted away from the WS stack and heap
  - Compiles to WS instruction set
  - Memory safe
- Assembly syntax language
  - Very close to WS instruction set
  - Assembly-like syntax
  - Allows additional metadata for compiler
  - Special compiler treatment of standard library functions
- Whitespace language
  - Direct representation of WS instruction set
  - Follows WS language spec syntax

## Naming

Nebula's name is derived from the stellar gas clouds, nebulae. Nebulae
are incredibly diffuse, yet form some of the most beautiful astronomical
landscapes. Similarly, of all programming languages, Whitespace is the
least dense in visible characters and despite its simplicity, is Turing
complete. The name is also a call-out to the star-forming region
[IC 1396](https://nitarp.ipac.caltech.edu/system/media_files/binaries/191/original/johnson2017sci.pdf)
that contains many nebulae including the famous Elephant's Trunk Nebula.
