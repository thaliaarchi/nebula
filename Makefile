all: build/99_bottles build/ascii build/ascii4 build/caesar \
	build/collatz build/factorial build/fib build/fizz_buzz \
	build/hello_world build/interpret build/pi

build/%: programs/%.ws nebula compile codegen/ext/ext.c
	./compile $< $@

build/%: programs/%.out.ws nebula compile codegen/ext/ext.c
	./compile $< $@

clean:
	rm -rf build
