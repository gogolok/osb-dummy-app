build: ## Builds the starter pack
	go build -i github.com/gogolok/osb-starter-pack/cmd/servicebroker

test: ## Runs the tests
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

clean: ## Cleans up build artifacts
	rm -f servicebroker

.PHONY: build test clean
