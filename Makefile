OPTS=GO111MODULE=on 
DOCKER_IMAGE=buildpack-deps:stretch-scm # docker image to use for running skywire-node. `golang` is OK too
DOCKER_NETWORK=SKYNET 
DOCKER_NODE=SKY01 


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

# Node

docker-clean: 
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_NODE} 

docker-network:
	-docker network create ${DOCKER_NETWORK}

docker-volume: build
	mkdir -p ./node 
	cp ./skywire-node ./node
	cp -r ./apps ./node/apps
	./skywire-cli config ./node/skywire.json
	cat ./node/skywire.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' ' > ./node/PK 

node: docker-clean docker-network docker-volume
	docker run -d -v $(shell pwd)/node:/sky --network=${DOCKER_NETWORK} --name=${DOCKER_NODE} ${DOCKER_IMAGE} bash -c "cd /sky && ./skywire-node"

run: 
	./skywire-node

node-stop:
	-docker container stop ${DOCKER_NODE}

refresh-node: node-stop
	cp ./skywire-node ./node
	docker container start  ${DOCKER_NODE}