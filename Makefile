all: build test checkdocs

install:
	go install

build:
	./build.sh

test:
	go test ./generators
	go test ./config
	go test ./tracker

checkdocs:
	./check_docs.sh

docs:
	cd docs_src && hugo

.PHONY: build docs
