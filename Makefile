.PHONY: build clean test

build:
	cd cmd/joc && go build -o ../../bin/joc

output/%.go: examples/%.jo
	@echo $<
	@./jo build $< > $@
	@diff -u $(basename $<).go $@
	@rm $@

test: clean build output $(patsubst examples/%.go,output/%.go,$(wildcard examples/*.go))

output:
	@mkdir -p output

clean:
	@rm -rf output
