all: build/99_bottles build/ascii build/ascii4 build/caesar \
	build/collatz build/factorial build/fib build/fizz_buzz \
	build/hello_world build/interpret build/pi \
	build/rosetta/add build/rosetta/binary build/rosetta/fib  \
	build/rosetta/fibrec build/rosetta/fizz_buzz build/rosetta/freq \
	build/rosetta/harshad build/rosetta/langstons_ant \
	build/rosetta/octal build/rosetta/shell_sort \
	build/rosetta/term_cursor build/rosetta/while

build/rosetta/%: programs/rosetta/%.ws nebula compile codegen/ext/ext.c
	@mkdir -p build/rosetta
	./compile $< $@

build/%: programs/%.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/%: programs/%.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@

# TODO nofold is needed for now because reducing `mul %0 -1` to `neg %0`
# does not replace mul in the stack push list with neg. Once stack
# updates are first class NIR instructions, -nofold can be removed.

build/interpret: programs/interpret.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@ -O3 "-heap=1000000 -nofold"

build/rosetta/langstons_ant: programs/rosetta/langstons_ant.ws nebula compile codegen/ext/ext.c
	@mkdir -p build/rosetta
	./compile $< $@ -O3 "-heap=10004 -nofold"

clean:
	rm -rf build
