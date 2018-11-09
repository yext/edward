all: build test

PKGS=`go list ./... | grep -v /vendor/ | grep -v /examples/`

version:
	./version.sh

install:
	go install

build:
	./build.sh

test: unit acceptance

unit:
	go test -timeout 3m -race -cover -count 1 $(PKGS)

acceptance:
	go test -timeout 3m -race -cover -count 1 github.com/yext/edward/test/acceptance -edward.acceptance 

docs:
	cd docs_src && hugo

release: version docs
	./release.sh

servedocs:
	cd docs_src && hugo serve

.PHONY: build docs version
