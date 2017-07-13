all: build test checkdocs

PKGS=`go list ./... | grep -v /vendor/ | grep -v /examples/`

install:
	go install

build:
	./build.sh

test:
	go test -race -cover $(PKGS)

checkdocs:
	./check_docs.sh

docs:
	cd docs_src && hugo

.PHONY: build docs
