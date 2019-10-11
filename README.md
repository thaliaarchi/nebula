# Wspace

Wspace is a compiler for the Whitespace programming language.

This project is evolving very rapidly, so documentation may not
represent its current state.

## Project Goals

### Compilation

- Static single assignment form
  - [x] Stack access converted to indexed variables
  - [x] Connected graph of basic blocks
  - [x] Minimize stack pointer updates to once per block
- Intraprocedural analysis
  - [x] Constant stack folding and propagation
  - [x] Constant stack string concatenation
- Interprocedural analysis
  - [x] Dead code elimination
  - [x] Call stack underflow detection
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
  - [ ] LLVM IR
  - [ ] WebAssembly

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
