OPTS?=GO111MODULE=on 
DOCKER_IMAGE?=skywire-runner # docker image to use for running skywire-node.`golang`, `buildpack-deps:stretch-scm`  is OK too
DOCKER_NETWORK?=SKYNET 
DOCKER_NODE?=SKY01
DOCKER_OPTS?=GO111MODULE=on GOOS=linux # go options for compiling for docker container

build: dep host-apps bin

clean:
	-rm -rf ./apps
	-rm -f ./skywire-node ./skywire-cli ./manager-node ./thereallssh-cli

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


# Apps 
host-apps: 
	${OPTS} go build -o ./apps/chat.v1.0 ./cmd/apps/chat	
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -o ./apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -o ./apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -o ./apps/therealssh.v1.0  ./cmd/apps/therealssh
	${OPTS} go build -o ./apps/therealssh-client.v1.0  ./cmd/apps/therealssh-client

# Bin 
bin: 
	${OPTS} go build -o ./skywire-node ./cmd/skywire-node 
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli 
	${OPTS} go build -o ./manager-node ./cmd/manager-node 
	${OPTS} go build -o ./therealssh-cli ./cmd/therealssh-cli

# Node

docker-image:
	docker image build --tag=skywire-runner --rm  - < skywire-runner.Dockerfile

docker-clean: 
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_NODE} 

docker-network:
	-docker network create ${DOCKER_NETWORK}

docker-apps:
	-${DOCKER_OPTS} go build -o ./node/apps/chat.v1.0 ./cmd/apps/chat
	-${DOCKER_OPTS} go build -o ./node/apps/helloworld.v1.0 ./cmd/apps/helloworld
	-${DOCKER_OPTS} go build -o ./node/apps/therealproxy.v1.0 ./cmd/apps/therealproxy
	-${DOCKER_OPTS} go build -o ./node/apps/therealproxy-client.v1.0  ./cmd/apps/therealproxy-client
	-${DOCKER_OPTS} go build -o ./node/apps/therealssh.v1.0  ./cmd/apps/therealssh
	-${DOCKER_OPTS} go build -o ./node/apps/therealssh-client.v1.0  ./cmd/apps/therealssh-client

docker-bin: 
	${DOCKER_OPTS} go build -o ./node/skywire-node ./cmd/skywire-node 


docker-volume: docker-apps docker-bin bin		
	./skywire-cli config ./node/skywire.json
	cat ./node/skywire.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' ' > ./node/PK 
	cat ./node/PK

node: docker-clean docker-image docker-network docker-volume 
	docker run -d -v $(shell pwd)/node:/sky --network=${DOCKER_NETWORK} --name=${DOCKER_NODE} ${DOCKER_IMAGE} bash -c "cd /sky && ./skywire-node"

node-stop:
	-docker container stop ${DOCKER_NODE}

refresh-node: node-stop docker-bin 
	# cp ./skywire-node ./node	
	docker container start  ${DOCKER_NODE}

# Host goals

run: stop build	
	cat ./skywire.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' ' > ./PK  
	./skywire-node &>/dev/null &

stop:
	-bash -c "kill $$(ps aux |grep '[s]kywire-node' |awk '{print $$2}')"

refresh: stop
	${OPTS} go build -o ./skywire-node ./cmd/skywire-node 
	./skywire-node &>/dev/null &
	
