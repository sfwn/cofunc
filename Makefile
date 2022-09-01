.PHONY: all build clean install
all:
	BUILD=true TEST=true cofunc run make.flowl

build:
	BUILD=true cofunc run make.flowl

test:
	TEST=true cofunc run make.flowl

clean:
	rm -rf bin/