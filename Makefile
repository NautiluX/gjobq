all:
	go build
example:
	make -C examples/simple
clean:
	make -C examples/simple clean
