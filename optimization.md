# Optimization strategies

## Instruction reductions
- Replace `call l; ret` with `jmp l`
- Replace `push n; push n` with `push n; dup`
- Replace `copy n; copy n+1` with `copy n; dup`
- Replace `copy 0` with `dup`
- Replace `swap; drop` with `slide 1`
- Remove `push n; drop`
- Remove `swap; swap`
- Replace n>2 successive drops with `slide n-1; drop`
- Remove uncalled labels
- Remove unreachable blocks after `jmp`, `ret`, `end`, or unreturned `jmp`
- Replace `call` that has no corresponding return with `jmp`
- Remove impossible conditional jumps
- Replace tautological conditional jumps with `jmp`
- Convert calls to be tail recursive
- Evaluate constant arithmetic at compile-time
- Remove stack values that are never used and dropped
- Inline blocks that have only one caller
- Reassign stack values after last usage

## Interpreter specializations
- Specialize arithmetic, jump, heap, and io instructions to not pop when preceded with `dup`
- Specialize instructions with non-arbitrary size arguments
- Consolidate successive stack underflow checks
- Replace calls to standard library with efficient interpreter operations
- Replace `mul` and `div` of powers of two with shl and shr

## Runtime analysis
- Insert profile points to identify slow points
- Track common branching
