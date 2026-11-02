BINARY := oprov
GOFLAGS ?=
GOMAXPROCS ?= 2

.PHONY: build test vet fmt install clean

build:
	GOMAXPROCS=$(GOMAXPROCS) go build $(GOFLAGS) -o $(BINARY) .

test:
	GOMAXPROCS=$(GOMAXPROCS) go test -count=1 ./...

vet:
	GOMAXPROCS=$(GOMAXPROCS) go vet ./...

fmt:
	gofmt -w .

install: build
	install -m755 $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
