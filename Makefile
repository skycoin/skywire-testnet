OPTS=GO111MODULE=on

build: dep apps bin

clean:
	rm -rf ./apps
	rm -f ./skywire-node ./skywire-cli ./manager-node ./thereallssh-cli

install:
	${OPTS} go install ./cmd/skywire-node ./cmd/skywire-cli ./cmd/manager-node ./cmd/therealssh-cli	

lint: ## Run linters. Use make install-linters first.
	# ${OPTS} vendorcheck ./... # TODO: fix vendor check
	${OPTS} golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	${OPTS} go vet -all ./...

install-linters: ## Install linters
	GO111MODULE=off go get -u github.com/FiloSottile/vendorcheck
	# For some reason this install method is not recommended, see https://github.com/golangci/golangci-lint#install
	# However, they suggest `curl ... | bash` which we should not do
	${OPTS} go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	${OPTS} go get -u golang.org/x/tools/cmd/goimports

format: ## Formats the code. Must have goimports installed (use make install-linters).
	${OPTS} goimports -w -local github.com/skycoin/skywire ./pkg
	${OPTS} goimports -w -local github.com/skycoin/skywire ./cmd
	${OPTS} goimports -w -local github.com/skycoin/skywire ./internal

dep: ## sorts dependencies
	${OPTS} go mod vendor -v

test: ## Run tests for net
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./internal/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/...


build: apps bin

# Apps 
apps: chat helloworld therealproxy therealproxy-client therealssh thereallssh-client

chat:
	${OPTS} go build -o ./apps/chat.v1.0 ./cmd/apps/chat

helloworld:
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld

therealproxy:
	${OPTS} go build -o ./apps/therealproxy.v1.0 ./cmd/apps/therealproxy

therealproxy-client:	
	${OPTS} go build -o ./apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client

therealssh:
	${OPTS} go build -o ./apps/therealssh.v1.0  ./cmd/apps/therealssh

thereallssh-client:
	${OPTS} go build -o ./apps/therealssh-client.v1.0  ./cmd/apps/therealssh-client

# Bin 
bin: skywire-node skywire-cli manager-node therealssh-cli

skywire-node:
	${OPTS} go build -o ./skywire-node ./cmd/skywire-node 

skywire-cli:
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli 

manager-node:
	${OPTS} go build -o ./manager-node ./cmd/manager-node 

therealssh-cli:
	${OPTS} go build -o ./therealssh-cli ./cmd/therealssh-cli
