#!/bin/bash

program="$1"
out="$2"

../nebula "$program" llvm 2> "$out.ll"

# llvm-link emits error "expected ')' at end of argument list" because of %0
sed -i '' 's/^declare void @printc(i64 %0)$/declare void @printc(i64)/' "$out.ll"
sed -i '' 's/^declare void @printi(i64 %0)$/declare void @printi(i64)/' "$out.ll"

clang -S -emit-llvm ../codegen/lib/io.c
llvm-link -o "$out.o" "$out.ll" io.ll
llc "$out.o"
clang -o "$out" "$out.o.s"
