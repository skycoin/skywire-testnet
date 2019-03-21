.DEFAULT_GOAL := help
.PHONY : check lint install-linters dep test 
.PHONY : build  clean install  format  
.PHONY : host-apps bin 
.PHONY : run stop 
.PHONY : docker-image  docker-clean docker-network  
.PHONY : docker-apps docker-bin docker-volume 
.PHONY : docker-run docker-stop     

OPTS?=GO111MODULE=on 
DOCKER_IMAGE?=skywire-runner # docker image to use for running skywire-node.`golang`, `buildpack-deps:stretch-scm`  is OK too
DOCKER_NETWORK?=SKYNET 
DOCKER_NODE?=SKY01
DOCKER_OPTS?=GO111MODULE=on GOOS=linux # go options for compiling for docker container

check: lint test ## Run linters and tests

build: dep host-apps bin ## Install dependencies, build apps and binaries. `go build` with ${OPTS} 

run: stop build	 ## Run skywire-node on host
	./skywire-node

stop: ## Stop running skywire-node on host
	-bash -c "kill $$(ps aux |grep '[s]kywire-node' |awk '{print $$2}')"


clean: ## Clean project: remove created binaries and apps
	-rm -rf ./apps
	-rm -f ./skywire-node ./skywire-cli ./manager-node ./thereallssh-cli

install: ## Install `skywire-node`, `skywire-cli`, `manager-node`, `therealssh-cli`
	${OPTS} go install ./cmd/skywire-node ./cmd/skywire-cli ./cmd/manager-node ./cmd/therealssh-cli	

lint: ## Run linters. Use make install-linters first
	# ${OPTS} vendorcheck ./... # TODO: fix vendor check
	${OPTS} golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	${OPTS} go vet -all ./...

test: ## Run tests for net
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./internal/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/...


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

dep: ## Sorts dependencies
	${OPTS} go mod vendor -v


# Apps 
host-apps: ## Build apps binaries 
	${OPTS} go build -o ./apps/chat.v1.0 ./cmd/apps/chat	
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -o ./apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -o ./apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -o ./apps/therealssh.v1.0  ./cmd/apps/therealssh
	${OPTS} go build -o ./apps/therealssh-client.v1.0  ./cmd/apps/therealssh-client

# Bin 
bin: ## Build `skywire-node`, `skywire-cli`, `manager-node`, `therealssh-cli`
	${OPTS} go build -o ./skywire-node ./cmd/skywire-node 
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli 
	${OPTS} go build -o ./manager-node ./cmd/manager-node 
	${OPTS} go build -o ./therealssh-cli ./cmd/therealssh-cli

# Dockerized skywire-node
docker-image: ## Build docker image `skywire-runner`
	docker image build --tag=skywire-runner --rm  - < skywire-runner.Dockerfile

docker-clean: ## Clean docker system: remove container ${DOCKER_NODE} and network ${DOCKER_NETWORK}
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_NODE} 

docker-network: ## Create docker network ${DOCKER_NETWORK}
	-docker network create ${DOCKER_NETWORK}

docker-apps: ## Build apps binaries for dockerized skywire-node. `go build` with  ${DOCKER_OPTS}
	-${DOCKER_OPTS} go build -o ./node/apps/chat.v1.0 ./cmd/apps/chat
	-${DOCKER_OPTS} go build -o ./node/apps/helloworld.v1.0 ./cmd/apps/helloworld
	-${DOCKER_OPTS} go build -o ./node/apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	-${DOCKER_OPTS} go build -o ./node/apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	-${DOCKER_OPTS} go build -o ./node/apps/therealssh.v1.0  ./cmd/apps/therealssh
	-${DOCKER_OPTS} go build -o ./node/apps/therealssh-client.v1.0  ./cmd/apps/therealssh-client

docker-bin: ## Build `skywire-node`, `skywire-cli`, `manager-node`, `therealssh-cli`. `go build` with  ${DOCKER_OPTS}
	${DOCKER_OPTS} go build -o ./node/skywire-node ./cmd/skywire-node 

docker-volume: docker-apps docker-bin bin  ## Prepare docker volume for dockerized skywire-node	
	./skywire-cli config ./node/skywire.json

docker-run: docker-clean docker-image docker-network docker-volume ## Run dockerized skywire-node ${DOCKER_NODE} in image ${DOCKER_IMAGE} with network ${DOCKER_NETWORK}
	docker run -it -v $(shell pwd)/node:/sky --network=${DOCKER_NETWORK} \
		--name=${DOCKER_NODE} ${DOCKER_IMAGE} bash -c "cd /sky && ./skywire-node"

docker-stop: ## Stop running dockerized skywire-node ${DOCKER_NODE}
	-docker container stop ${DOCKER_NODE}


help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	