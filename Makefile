GOCMD := go
GOFMT := ${GOCMD} fmt
GOMOD := ${GOCMD} mod

.PHONY: help
help: ## print this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-z0-9A-Z_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m%s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## tidy modules
	${GOMOD} tidy

.PHONY: fmt
fmt: ## apply go code style formatter
	${GOFMT} -x ./...

.PHONY: lint
lint: ## run linters
	golangci-lint run -v
	${GOCMD} tool nilaway ./...

.PHONY: test
test: fmt tidy ## run unit tests
	${GOCMD} test -race -v ./...

.PHONY: binary
binary: fmt tidy lint test ## build a binary via goreleaser
	goreleaser build --clean --single-target --snapshot --output .

.PHONY: build
build: binary ## alias for `binary`
