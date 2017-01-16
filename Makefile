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
	./push_docs.sh
