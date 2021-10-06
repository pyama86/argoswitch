INFO_COLOR=\033[1;34m
RESET=\033[0m
BOLD=\033[1m
GO ?= GO111MODULE=on go
TEST ?= $(shell $(GO) list ./... | grep -v -e vendor -e keys -e tmp)

VERSION = $(shell git describe --tags --abbrev=0)

.PHONY: run
run:
	go run main.go
.PHONY: devdeps
devdeps:
	go get github.com/codegangsta/gin
	which golint > /dev/null || go get -u golang.org/x/lint/golint
	which staticcheck > /dev/null || go get honnef.co/go/tools/cmd/staticcheck

.PHONY: test
test:
	@echo "$(INFO_COLOR)==> $(RESET)$(BOLD)Testing$(RESET)"
	$(GO) test $(TEST) -timeout=60s -parallel=4
	$(GO) test -race $(TEST)

.PHONY: lint
## lint: run golint
lint: devdeps
	@echo "$(INFO_COLOR)==> $(RESET)$(BOLD)Linting$(RESET)"
	golint -min_confidence 1.1 -set_exit_status $(TEST)
	staticcheck ./...

.PHONY: build
build: ## Build server
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o argoswitch


