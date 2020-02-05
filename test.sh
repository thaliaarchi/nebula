#!/bin/bash

# Reference interpreter path
wspace=wspace

make all

echo "99_bottles:";  diff <(build/99_bottles)  <("$wspace" programs/99_bottles.out.ws)
echo "ascii:";       diff <(build/ascii)       <("$wspace" programs/ascii.out.ws)
echo "ascii4:";      diff <(build/ascii4)      <("$wspace" programs/ascii4.out.ws)
echo "fizz_buzz:";   diff <(build/fizz_buzz)   <("$wspace" programs/fizz_buzz.out.ws)
echo "hello_world:"; diff <(build/hello_world) <("$wspace" programs/hello_world.ws)

echo "collatz 5:";    diff <(echo 5   | build/collatz)   <(echo 5   | "$wspace" programs/collatz.out.ws)
echo "collatz 10:";   diff <(echo 10  | build/collatz)   <(echo 10  | "$wspace" programs/collatz.out.ws)
echo "collatz 100:";  diff <(echo 100 | build/collatz)   <(echo 100 | "$wspace" programs/collatz.out.ws)
echo "factorial 10:"; diff <(echo 10  | build/factorial) <(echo 10  | "$wspace" programs/factorial.out.ws)
echo "pi 5:";         diff <(echo 5   | build/pi)        <(echo 5   | "$wspace" programs/pi.out.ws)
