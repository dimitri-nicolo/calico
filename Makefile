.PHONY: all binary test clean help
all: dist/carrotctl dist/carrotctl-darwin-amd64 dist/carrotctl-windows-amd64.exe test
test:
	go test ./...

vendor: Gopkg.toml
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

CARROTCTL_VERSION?=$(shell git describe --tags --dirty --always)
CARROTCTL_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CARROTCTL_GIT_REVISION?=$(shell git rev-parse --short HEAD)


LDFLAGS=-ldflags "-X github.com/tigera/licensing/carrotctl/cmd.VERSION=$(CARROTCTL_VERSION) \
	-X github.com/tigera/licensing/carrotctl/cmd.BUILD_DATE=$(CARROTCTL_BUILD_DATE) \
	-X github.com/tigera/licensing/carrotctl/cmd.GIT_REVISION=$(CARROTCTL_GIT_REVISION)"
	
## Build carrotctl
build: vendor dist/carrotctl dist/carrotctl-linux-amd64 dist/carrotctl-darwin-amd64 dist/carrotctl-windows-amd64.exe

dist/carrotctl: vendor dist/carrotctl-linux-amd64
	cp dist/carrotctl-linux-amd64 dist/carrotctl

dist/carrotctl-linux-amd64: vendor
	GOOS=linux GOARCH=amd64 go build -o dist/carrotctl-linux-amd64 $(LDFLAGS) "./carrotctl/carrotctl.go"

dist/carrotctl-darwin-amd64: vendor
	GOOS=darwin GOARCH=amd64 go build -o dist/carrotctl-darwin-amd64 $(LDFLAGS) "./carrotctl/carrotctl.go"

dist/carrotctl-windows-amd64.exe: vendor
	GOOS=windows GOARCH=amd64 go build -o dist/carrotctl-windows-amd64.exe $(LDFLAGS) "./carrotctl/carrotctl.go"

.PHONY: install
install:
	CGO_ENABLED=0 go install github.com/tigera/licensing/carrotctl

## Clean enough that a new release build will be clean
clean:
	rm -rf dist build certs *.tar vendor