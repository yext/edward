all: test

install:
	go install

build:
	go build

test:
	go test ./generators
	go test ./config

docs:
	./push_docs.sh
