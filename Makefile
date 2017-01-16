all: test

install:
	go install

build:
	go build

test:
	go test ./generators
	go test ./config

docs:
	cd docs_site && hugo

pushdocs:
	git subtree push --prefix docs_site/public origin gh-pages
