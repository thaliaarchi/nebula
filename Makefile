all: build/99_bottles build/ascii build/ascii4 build/caesar \
	build/collatz build/factorial build/fib build/fizz_buzz \
	build/hello_world build/interpret build/pi

build/%: programs/%.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/%: programs/%.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/interpret: programs/interpret.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@ -O3 -heap=1000000

clean:
	rm -rf build
