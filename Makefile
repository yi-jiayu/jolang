.PHONY: build clean test

build:
	cd cmd/joc && go build -o ../../bin/joc

output/%.go: tests/%.jo
	@echo $<
	@./jo build $< > $@
	@diff -u $(basename $<).go $@
	@rm $@

test: clean build output $(patsubst tests/%.go,output/%.go,$(wildcard tests/*.go))

output:
	@mkdir -p output

clean:
	@rm -rf output
