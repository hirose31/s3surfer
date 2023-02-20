BIN := s3surfer
MAIN = ./cmd/s3surfer
VERSION := $$(make -s show-version)
CURRENT_REVISION := $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS := "-s -w -X main.revision=$(CURRENT_REVISION)"
GOBIN ?= $(shell go env GOPATH)/bin
u := $(if $(update),-u) # make update=1 deps

.PHONY: help
.DEFAULT_GOAL := help

help:
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: clean build ## clean and build

.PHONY: build
build: deps ## build
	go build -ldflags=$(BUILD_LDFLAGS) -o $(BIN) $(MAIN)

.PHONY: install
install: deps ## install
	go install -ldflags=$(BUILD_LDFLAGS) $(MAIN)

.PHONY: clean
clean: ## clean
	rm -rf $(BIN) goxz
	go clean

.PHONY: test
test: deps ## test
	go test -v ./...

.PHONY: lint
lint: devel-deps ## run golint and staticcheck
	golint -set_exit_status ./...
	staticcheck -checks all ./...

.PHONY: security
security: devel-deps ## run gosec
	gosec ./...

.PHONY: bump
bump: devel-deps  ## release new version
ifneq ($(shell git status --porcelain),)
	$(error git workspace is dirty)
endif
ifneq ($(shell git rev-parse --abbrev-ref HEAD),master)
	$(error current branch is not master)
endif
	@gobump up -w $(MAIN)
	ghch -w -N "v$(VERSION)"
	git commit -am "bump up version to $(VERSION)"
	git tag "v$(VERSION)"
	git push origin master
	git push origin "refs/tags/v$(VERSION)"

.PHONY: cross
cross: devel-deps ## build for cross platforms
	goxz -arch amd64,arm64 -os linux,darwin -n $(BIN) -pv=v$(VERSION) -build-ldflags=$(BUILD_LDFLAGS) -trimpath .
	goxz -arch amd64       -os windows      -n $(BIN) -pv=v$(VERSION) -build-ldflags=$(BUILD_LDFLAGS) -trimpath .

.PHONY: upload
upload: devel-deps ## upload
	ghr "v$(VERSION)" goxz/


.PHONY: show-version
show-version: devel-deps ## show-version
	@gobump show -r $(MAIN)

.PHONY: deps
deps:
	go get ${u} -d -v $(MAIN)
	go mod tidy

.PHONY: devel-deps
devel-deps: $(GOBIN)/golint $(GOBIN)/staticcheck $(GOBIN)/gosec $(GOBIN)/gobump $(GOBIN)/ghch $(GOBIN)/ghr $(GOBIN)/goxz

$(GOBIN)/golint:
	go install golang.org/x/lint/golint@latest

$(GOBIN)/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

$(GOBIN)/gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest

$(GOBIN)/gobump:
	go install github.com/x-motemen/gobump/cmd/gobump@latest

$(GOBIN)/ghch:
	go install github.com/Songmu/ghch/cmd/ghch@latest

$(GOBIN)/ghr:
	go install github.com/tcnksm/ghr@latest

$(GOBIN)/goxz:
	go install github.com/Songmu/goxz/cmd/goxz@latest


