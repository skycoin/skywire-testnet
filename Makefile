lint: ## Run linters. Use make install-linters first.
	GO111MODULE=on vendorcheck ./...
	GO111MODULE=on golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	GO111MODULE=on go vet -all ./...

install-linters: ## Install linters
	GO111MODULE=on go get -u github.com/FiloSottile/vendorcheck
	# For some reason this install method is not recommended, see https://github.com/golangci/golangci-lint#install
	# However, they suggest `curl ... | bash` which we should not do
	GO111MODULE=on go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

format: ## Formats the code. Must have goimports installed (use make install-linters).
	GO111MODULE=on goimports -w -local github.com/skycoin/skywire ./pkg
	GO111MODULE=on goimports -w -local github.com/skycoin/skywire ./cmd
	GO111MODULE=on goimports -w -local github.com/skycoin/skywire ./internal

dep: ## sorts dependencies
	GO111MODULE=on go mod vendor -v

test: ## Run tests for net
	GO111MODULE=on go test -race -tags no_ci -cover -timeout=5m ./internal/...
	GO111MODULE=on go test -race -tags no_ci -cover -timeout=5m ./pkg/...
