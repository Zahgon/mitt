.PHONY: check build vet fmt lint test race cover bench clean

check: build vet fmt lint race cover

build:
	go build ./...

vet:
	go vet ./...

fmt:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Not gofmt'd:"; echo "$$unformatted"; gofmt -d .; exit 1; \
	fi

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping"; \
	fi

test:
	go test ./...

race:
	go test ./... -race

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

bench:
	go test ./... -run '^$$' -bench . -benchmem

clean:
	rm -f coverage.out
