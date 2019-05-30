.DEFAULT_GOAL := help
.PHONY : check lint install-linters dep test 
.PHONY : build  clean install  format  
.PHONY : host-apps bin 
.PHONY : run stop config
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

run: stop build	config  ## Run skywire-node on host
	./skywire-node skywire.json

stop: ## Stop running skywire-node on host
	-bash -c "kill $$(ps aux |grep '[s]kywire-node' |awk '{print $$2}')"

config: ## Generate skywire.json
	-./skywire-cli node gen-config -o  ./skywire.json -r

clean: ## Clean project: remove created binaries and apps
	-rm -rf ./apps
	-rm -f ./skywire-node ./skywire-cli ./setup-node ./manager-node ./skywire-messenger-ssh-cli

install: ## Install `skywire-node`, `skywire-skywire-messenger-ssh-cli`, `manager-node`, `skywire-messenger-skywire-messenger-ssh-cli`
	${OPTS} go install ./cmd/skywire-node ./cmd/skywire-cli ./cmd/setup-node ./cmd/manager-node ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-cli	

rerun: stop
	${OPTS} go build -race -o ./skywire-node ./cmd/skywire-node 
	-./skywire-cli node gen-config -o  ./skywire.json -r
	perl -pi -e 's/localhost//g' ./skywire.json
	./skywire-node skywire.json


lint: ## Run linters. Use make install-linters first	
	${OPTS} golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	${OPTS} go vet -all ./...

vendorcheck:  ## Run vendorcheck
	GO111MODULE=off vendorcheck ./skywire-messenger-ssh/... 
	GO111MODULE=off vendorcheck ./internal/... 
	GO111MODULE=off vendorcheck ./pkg/... 
	GO111MODULE=off vendorcheck ./cmd/apps/... 
	GO111MODULE=off vendorcheck ./cmd/manager-node/... 
	GO111MODULE=off vendorcheck ./cmd/setup-node/... 
	GO111MODULE=off vendorcheck ./cmd/skywire-cli/... 
	GO111MODULE=off vendorcheck ./cmd/skywire-node/... 
	# vendorcheck fails on ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-cli/-skywire-messenger-ssh-cli
	# the problem is indirect dependency to github.com/sirupsen/logrus
	#GO111MODULE=off vendorcheck ./cmd/-skywire-messenger-ssh-cli/...

test: ## Run tests
	-go clean -testcache &>/dev/null
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./internal/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./skywire-messenger-ssh/...
	#${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/app/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/cipher/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/manager/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/messaging-discovery/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/node/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/route-finder/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/router/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/routing/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/setup/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/transport/...
	${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/transport-discovery/...
	${OPTS} go test  -tags no_ci -cover -timeout=5m ./pkg/messaging/...


install-linters: ## Install linters
	- VERSION=1.13.2 ./ci_scripts/install-golangci-lint.sh 
	# GO111MODULE=off go get -u github.com/FiloSottile/vendorcheck
	# For some reason this install method is not recommended, see https://github.com/golangci/golangci-lint#install
	# However, they suggest `curl ... | bash` which we should not do
	# ${OPTS} go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	${OPTS} go get -u golang.org/x/tools/cmd/goimports

format: ## Formats the code. Must have goimports installed (use make install-linters).
	${OPTS} goimports -w -local github.com/skycoin/skywire ./pkg
	${OPTS} goimports -w -local github.com/skycoin/skywire ./cmd
	${OPTS} goimports -w -local github.com/skycoin/skywire ./internal
	${OPTS} goimports -w -local github.com/skycoin/skywire ./skywire-messenger-ssh

dep: ## Sorts dependencies
	${OPTS} go mod vendor -v

# Apps 
host-apps: ## Build app 
	${OPTS} go build -race -o ./apps/chat.v1.0 ./cmd/apps/chat	
	${OPTS} go build -race -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -race -o ./apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -race -o ./apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -race -o ./apps/skywire-messenger-ssh-server.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-server
	${OPTS} go build -race -o ./apps/skywire-messenger-ssh-client.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-client

# Bin 
bin: ## Build `skywire-node`, `skywire-cli`, `setup-node`,`manager-node`, `skywire-messenger-ssh-cli`
	${OPTS} go build -race -o ./skywire-node ./cmd/skywire-node 
	${OPTS} go build -race -o ./skywire-cli  ./cmd/skywire-cli 
	${OPTS} go build -race -o ./setup-node ./cmd/setup-node
	${OPTS} go build -race -o ./manager-node ./cmd/manager-node 
	${OPTS} go build -race -o ./skywire-messenger-ssh-cli ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-cli

release: ## Build skywire-node`, skywire-cli, manager-node, skywire-messenger-ssh-cli and apps without -race flag
	${OPTS} go build -o ./skywire-node ./cmd/skywire-node 
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli 
	${OPTS} go build -o ./setup-node ./cmd/setup-node
	${OPTS} go build -o ./manager-node ./cmd/manager-node 
	${OPTS} go build -o ./skywire-messenger-ssh-cli ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-cli
	${OPTS} go build -o ./apps/chat.v1.0 ./cmd/apps/chat	
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -o ./apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -o ./apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -o ./apps/skywire-messenger-ssh-server.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-server
	${OPTS} go build -o ./apps/skywire-messenger-ssh-client.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-client



# Dockerized skywire-node
docker-image: ## Build docker image `skywire-runner`
	docker image build --tag=skywire-runner --rm  - < skywire-runner.Dockerfile

docker-clean: ## Clean docker system: remove container ${DOCKER_NODE} and network ${DOCKER_NETWORK}
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_NODE} 

docker-network: ## Create docker network ${DOCKER_NETWORK}
	-docker network create ${DOCKER_NETWORK}

docker-apps: ## Build apps binaries for dockerized skywire-node. `go build` with  ${DOCKER_OPTS}
	-${DOCKER_OPTS} go build -race -o ./node/apps/chat.v1.0 ./cmd/apps/chat
	-${DOCKER_OPTS} go build -race -o ./node/apps/helloworld.v1.0 ./cmd/apps/helloworld
	-${DOCKER_OPTS} go build -race -o ./node/apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	-${DOCKER_OPTS} go build -race -o ./node/apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	-${DOCKER_OPTS} go build -race -o ./node/apps/skywire-messenger-ssh-server.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-server
	-${DOCKER_OPTS} go build -race -o ./node/apps/skywire-messenger-ssh-client.v1.0  ./skywire-messenger-ssh/cmd/skywire-messenger-ssh-client

docker-bin: ## Build `skywire-node`. `go build` with  ${DOCKER_OPTS}
	${DOCKER_OPTS} go build -race -o ./node/skywire-node ./cmd/skywire-node 

docker-volume: dep docker-apps docker-bin bin  ## Prepare docker volume for dockerized skywire-node	
	-${DOCKER_OPTS} go build  -o ./docker/skywire-services/setup-node ./cmd/setup-node
	-./skywire-cli node gen-config -o  ./node/skywire.json -r
	perl -pi -e 's/localhost//g' ./node/skywire.json # To make node accessible from outside with skywire-cli

docker-run: docker-clean docker-image docker-network docker-volume ## Run dockerized skywire-node ${DOCKER_NODE} in image ${DOCKER_IMAGE} with network ${DOCKER_NETWORK}
	docker run -it -v $(shell pwd)/node:/sky --network=${DOCKER_NETWORK} \
		--name=${DOCKER_NODE} ${DOCKER_IMAGE} bash -c "cd /sky && ./skywire-node skywire.json"

docker-setup-node:	## Runs setup-node in detached state in ${DOCKER_NETWORK}
	-docker container rm setup-node -f
	docker run -d --network=${DOCKER_NETWORK}  	\
	 				--name=setup-node	\
	 				--hostname=setup-node	skywire-services \
					  bash -c "./setup-node setup-node.json"

docker-stop: ## Stop running dockerized skywire-node ${DOCKER_NODE}
	-docker container stop ${DOCKER_NODE}

docker-rerun: docker-stop
	-./skywire-cli gen-config -o  ./node/skywire.json -r
	perl -pi -e 's/localhost//g' ./node/skywire.json # To make node accessible from outside with skywire-cli
	${DOCKER_OPTS} go build -race -o ./node/skywire-node ./cmd/skywire-node 
	docker container start -i ${DOCKER_NODE}


help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	
