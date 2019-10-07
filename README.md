# Whitespace Goals

## Languages

- High level language
  - Abstracted away from WS
  - Compiles to WS instruction set
  - Memory safe
- Assembly syntax language
  - Very close to WS instruction set
  - Assembly-like syntax
  - Allows additional metadata for compiler
- Whitespace language
  - Direct representation of WS instruction set
  - Follows WS language spec syntax

## Instruction sets

- Whitespace
  - Core WS language instructions from spec
- Extended WS
  - Adds bitwise operations not defined by WS language
  - Pure WS fallbacks defined in std lib

## Intermediate representations

- WS IR
  - Tail call optimization
  - Connected basic blocks
- Atomic WS IR
  - Separates push, pop, and stack assertions from instructions
  - Dead code removal
  - Reduction of arbitrary precision integers
  - Connected basic blocks
- Register WS IR
  - Convert to register based
  - Minimal register coloring
  - SSA or CPS form
  - Connected basic blocks
- LLVM IR
  - Machine code compilation

## What I want to learn from Whitespace

- Stack-based to register-based
- Register coloring
- SSA or CPS intermediate representation
- Reduction of arbitrary precision integers
- Special compiler treatment of std lib calls
- Machine code compilation using LLVM IR
- Tail call optimization
- Lifetime analysis
- Constant folding and propagation
- Using an SMT solver
- Abstract interpretation
