.PHONY: fmt check-fmt lint build clean install

BINARY=GitEfFive

fmt:
	gofmt -w .

check-fmt:
	test -z "$$(gofmt -l .)"

lint:
	go vet ./...

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)

all: check-fmt lint build
