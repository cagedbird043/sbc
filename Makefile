.PHONY: build clean install

# Build flags
VERSION ?= dev
LDFLAGS = -X github.com/cagedbird043/sbc/cmd.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o sbc .

clean:
	rm -f sbc

install: build
	install -m 755 sbc /usr/local/bin/sbc
