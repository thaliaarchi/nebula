#!/bin/bash

# Reference interpreter path
wspace=wspace

make all

function compare() {
  binary="$1"
  program="$2"
  echo "$binary:"
  diff <("build/$binary") <("$wspace" "programs/$program")
}

function compare_arg() {
  binary="$1"
  program="$2"
  args="$3"
  echo "$binary $args:"
  diff <(echo "$args" | "build/$binary") <(echo "$args" | "$wspace" "programs/$program")
}

function compare_file() {
  binary="$1"
  program="$2"
  file="$3"
  echo "$binary $file:"
  diff <("build/$binary" < "$file") <("$wspace" "programs/$program" < "$file")
}

function compare_head() {
  binary="$1"
  program="$2"
  n="$3"
  echo "$binary $n:"
  diff <("build/$binary" | head -n "$n") <("$wspace" "programs/$program" | head -n "$n")
}

compare      99_bottles            99_bottles.out.ws
compare      ascii                 ascii.out.ws
compare      ascii4                ascii4.out.ws
compare_arg  collatz               collatz.out.ws           5
compare_arg  collatz               collatz.out.ws           10
compare_arg  collatz               collatz.out.ws           100
compare_arg  factorial             factorial.out.ws         10
compare      fizz_buzz             fizz_buzz.out.ws
compare      hello_world           hello_world.ws
compare_arg  pi                    pi.out.ws                5

compare_arg  rosetta/add           rosetta/add.ws           "42
314"
compare_head rosetta/binary        rosetta/binary.ws        100
compare_head rosetta/fib           rosetta/fib.ws           47
compare_arg  rosetta/fibrec        rosetta/fibrec.ws        20
compare      rosetta/fizz_buzz     rosetta/fizz_buzz.ws
compare      rosetta/harshad       rosetta/harshad.ws
compare_head rosetta/octal         rosetta/octal.ws         100
compare_file rosetta/shell_sort    rosetta/shell_sort.ws    programs/rosetta/shell_sort.in
compare      rosetta/term_cursor   rosetta/term_cursor.ws
compare      rosetta/while         rosetta/while.ws

compare      rosetta/langstons_ant rosetta/langstons_ant.ws
compare_file rosetta/freq          rosetta/freq.ws          programs/rosetta/freq.ws

cp programs/interpret.out.ws tmp.ws
printf '\0' >> tmp.ws
cat programs/hello_world.ws >> tmp.ws
printf '\0' >> tmp.ws

echo "hello_world:"
time build/interpret < tmp.ws
time "$wspace" programs/interpret.out.ws < tmp.ws
rm tmp.ws
