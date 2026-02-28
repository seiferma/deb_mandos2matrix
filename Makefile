APPNAME:=mandos2matrix
RELEASE?=0
CGO_ENABLED?=0

ifeq ($(RELEASE), 1)
	# Strip debug information from the binary
	GO_LDFLAGS+=-s -w
endif
GO_LDFLAGS:=-ldflags="$(GO_LDFLAGS)"
GO_BUILDFLAGS:=-tags goolm

.PHONY: default
default: test

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_LDFLAGS) $(GO_BUILDFLAGS) -o ./build/$(APPNAME) -v cmd/main.go


.PHONY: test
test: build
	CGO_ENABLED=$(CGO_ENABLED) go test $(GO_BUILDFLAGS) -v ./...

.PHONY: clean
clean:
	rm -rf ./build