.DEFAULT_GOAL := check

.PHONY: check fix tidy tidy-check generate fmt fmt-check vet lint lint-fix test test-race bench-all cover-html

# Verification never rewrites source, module files, or coverage artifacts.
check: tidy-check fmt-check vet lint test-race

# Explicitly apply dependency, generation, formatting, and lint fixes in order.
fix:
	$(MAKE) tidy
	$(MAKE) generate
	$(MAKE) fmt
	$(MAKE) lint-fix

tidy:
	go mod tidy

tidy-check:
	go mod tidy -diff

generate:
	go generate ./...

fmt:
	golangci-lint fmt

fmt-check:
	golangci-lint fmt --diff

lint:
	golangci-lint run --timeout=5m ./...

lint-fix:
	golangci-lint run --fix --timeout=5m ./...

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race -count=5 ./...

bench-all:
	go test -bench=. -benchmem ./...

cover-html:
	go test -coverprofile=./coverage.text -covermode=atomic ./...
	go tool cover -html=./coverage.text -o ./cover.html
	rm ./coverage.text
