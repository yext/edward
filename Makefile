all: build test checkdocs

PKGS=`go list ./... | grep -v /vendor/ | grep -v /examples/`

install:
	go install

build:
	./build.sh

test: unit acceptance

unit:
	go test -timeout 3m -race -cover -count 1 $(PKGS)

acceptance:
	go test -timeout 3m -race -cover -count 1 github.com/yext/edward/test/acceptance -edward.acceptance 

checkdocs:
	./check_docs.sh

docs:
	cd docs_src && hugo

servedocs:
	cd docs_src && hugo serve

.PHONY: build docs
