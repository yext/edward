all: build test

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

docs:
	cd docs_src && hugo

release: docs
	git checkout master
	git pull
	git merge develop
	git tag v1.8.14
	git push origin v1.8.14

servedocs:
	cd docs_src && hugo serve

.PHONY: build docs
