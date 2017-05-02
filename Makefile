all: build test checkdocs

install:
	go install

build:
	./build.sh

test:
	go test -race -cover ./generators
	go test -race -cover ./config
	go test -race -cover ./tracker

checkdocs:
	./check_docs.sh

docs:
	cd docs_src && hugo

.PHONY: build docs
