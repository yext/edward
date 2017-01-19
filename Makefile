all: build test checkdocs

install:
	go install

build:
	go build

test:
	go test ./generators
	go test ./config

checkdocs:
	./check_docs.sh

docs:
	cd docs_src && hugo
