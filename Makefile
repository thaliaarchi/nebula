all: build/99_bottles build/ascii build/ascii4 build/caesar \
	build/collatz build/factorial build/fib build/fizz_buzz \
	build/hello_world build/interpret build/pi \
	build/rosetta/add build/rosetta/binary build/rosetta/fib  \
	build/rosetta/fibrec build/rosetta/fizz_buzz build/rosetta/freq \
	build/rosetta/harshad build/rosetta/langstons_ant \
	build/rosetta/octal build/rosetta/shell_sort \
	build/rosetta/term_cursor build/rosetta/while \
	build/test_ret_underflow

build/rosetta/%: programs/rosetta/%.ws nebula compile codegen/ext/ext.c
	@mkdir -p build/rosetta
	./compile $< $@

build/%: programs/%.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/%: programs/%.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/interpret: programs/interpret.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@ -O3 -heap=1000000

build/rosetta/langstons_ant: programs/rosetta/langstons_ant.ws nebula compile codegen/ext/ext.c
	@mkdir -p build/rosetta
	./compile $< $@ -O3 -heap=10004

clean:
	rm -rf build
