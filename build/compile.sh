#!/bin/bash

program="$1"
out="$2"

../nebula "$program" llvm > "$out.ll"
clang -S -emit-llvm ../codegen/lib/io.c
llvm-link -o "$out.o" "$out.ll" io.ll
llc "$out.o"
clang -o "$out" "$out.o.s"