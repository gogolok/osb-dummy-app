build: ## Builds the dummy app
	go build github.com/gogolok/osb-dummy-app/cmd/servicebroker

test: ## Runs the tests
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

clean: ## Cleans up build artifacts
	rm -f servicebroker

.PHONY: build test clean
