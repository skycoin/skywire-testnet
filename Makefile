.DEFAULT_GOAL := help
.PHONY : check lint install-linters dep test 
.PHONY : build  clean install  format  bin
.PHONY : host-apps bin 
.PHONY : run stop config
.PHONY : docker-image  docker-clean docker-network  
.PHONY : docker-apps docker-bin docker-volume 
.PHONY : docker-run docker-stop     

OPTS?=GO111MODULE=on
DOCKER_IMAGE?=skywire-runner # docker image to use for running skywire-visor.`golang`, `buildpack-deps:stretch-scm`  is OK too
DOCKER_NETWORK?=SKYNET 
DOCKER_NODE?=SKY01
DOCKER_OPTS?=GO111MODULE=on GOOS=linux # go options for compiling for docker container
TEST_OPTS?=-race -tags no_ci -cover -timeout=5m
BUILD_OPTS?=-race

check: lint test ## Run linters and tests

build: dep host-apps bin ## Install dependencies, build apps and binaries. `go build` with ${OPTS} 

run: stop build	config  ## Run skywire-visor on host
	./skywire-visor skywire.json

stop: ## Stop running skywire-visor on host
	-bash -c "kill $$(ps aux |grep '[s]kywire-visor' |awk '{print $$2}')"

config: ## Generate skywire.json
	-./skywire-cli node gen-config -o  ./skywire.json -r

clean: ## Clean project: remove created binaries and apps
	-rm -rf ./apps
	-rm -f ./skywire-visor ./skywire-cli ./setup-node ./hypervisor ./SSH-cli

install: ## Install `skywire-visor`, `skywire-cli`, `hypervisor`, `SSH-cli`
	${OPTS} go install ./cmd/skywire-visor ./cmd/skywire-cli ./cmd/setup-node ./cmd/hypervisor ./cmd/therealssh-cli

rerun: stop
	${OPTS} go build -race -o ./skywire-visor ./cmd/skywire-visor
	-./skywire-cli node gen-config -o  ./skywire.json -r
	perl -pi -e 's/localhost//g' ./skywire.json
	./skywire-visor skywire.json


lint: ## Run linters. Use make install-linters first	
	${OPTS} golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	${OPTS} go vet -mod=vendor -all ./...

vendorcheck:  ## Run vendorcheck
	GO111MODULE=off vendorcheck ./internal/... 
	GO111MODULE=off vendorcheck ./pkg/... 
	GO111MODULE=off vendorcheck ./cmd/apps/... 
	GO111MODULE=off vendorcheck ./cmd/hypervisor/...
	GO111MODULE=off vendorcheck ./cmd/setup-node/... 
	GO111MODULE=off vendorcheck ./cmd/skywire-cli/... 
	GO111MODULE=off vendorcheck ./cmd/skywire-visor/...
	# vendorcheck fails on ./cmd/therealssh-cli
	# the problem is indirect dependency to github.com/sirupsen/logrus
	#GO111MODULE=off vendorcheck ./cmd/therealssh-cli/... 	

test: ## Run tests
	-go clean -testcache &>/dev/null
	${OPTS} go test ${TEST_OPTS} ./internal/...
	${OPTS} go test ${TEST_OPTS} ./pkg/...

install-linters: ## Install linters
	- VERSION=1.17.1 ./ci_scripts/install-golangci-lint.sh 
	# GO111MODULE=off go get -u github.com/FiloSottile/vendorcheck
	# For some reason this install method is not recommended, see https://github.com/golangci/golangci-lint#install
	# However, they suggest `curl ... | bash` which we should not do
	# ${OPTS} go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	${OPTS} go get -u golang.org/x/tools/cmd/goimports

format: ## Formats the code. Must have goimports installed (use make install-linters).
	${OPTS} goimports -w -local github.com/skycoin/skywire ./pkg
	${OPTS} goimports -w -local github.com/skycoin/skywire ./cmd
	${OPTS} goimports -w -local github.com/skycoin/skywire ./internal

dep: ## Sorts dependencies
	${OPTS} go mod vendor -v

# Apps 
host-apps: ## Build app 
	${OPTS} go build ${BUILD_OPTS} -o ./apps/skychat.v1.0 ./cmd/apps/skychat	
	${OPTS} go build ${BUILD_OPTS} -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build ${BUILD_OPTS} -o ./apps/socksproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build ${BUILD_OPTS} -o ./apps/socksproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build ${BUILD_OPTS} -o ./apps/SSH.v1.0  ./cmd/apps/therealssh
	${OPTS} go build ${BUILD_OPTS} -o ./apps/SSH-client.v1.0  ./cmd/apps/therealssh-client

# Bin 
bin: ## Build `skywire-visor`, `skywire-cli`, `hypervisor`, `SSH-cli`
	${OPTS} go build ${BUILD_OPTS} -o ./skywire-visor ./cmd/skywire-visor
	${OPTS} go build ${BUILD_OPTS} -o ./skywire-cli  ./cmd/skywire-cli
	${OPTS} go build ${BUILD_OPTS} -o ./setup-node ./cmd/setup-node
	${OPTS} go build ${BUILD_OPTS} -o ./messaging-server ./cmd/messaging-server
	${OPTS} go build ${BUILD_OPTS} -o ./hypervisor ./cmd/hypervisor
	${OPTS} go build ${BUILD_OPTS} -o ./SSH-cli ./cmd/therealssh-cli


release: ## Build `skywire-visor`, `skywire-cli`, `hypervisor`, `SSH-cli` and apps without -race flag
	${OPTS} go build -o ./skywire-visor ./cmd/skywire-visor
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli
	${OPTS} go build -o ./setup-node ./cmd/setup-node
	${OPTS} go build -o ./hypervisor ./cmd/hypervisor
	${OPTS} go build -o ./SSH-cli ./cmd/therealssh-cli
	${OPTS} go build -o ./apps/skychat.v1.0 ./cmd/apps/skychat
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -o ./apps/socksproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -o ./apps/socksproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -o ./apps/SSH.v1.0  ./cmd/apps/therealssh
	${OPTS} go build -o ./apps/SSH-client.v1.0  ./cmd/apps/therealssh-client

# Dockerized skywire-visor
docker-image: ## Build docker image `skywire-runner`
	docker image build --tag=skywire-runner --rm  - < skywire-runner.Dockerfile

docker-clean: ## Clean docker system: remove container ${DOCKER_NODE} and network ${DOCKER_NETWORK}
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_NODE}

docker-network: ## Create docker network ${DOCKER_NETWORK}
	-docker network create ${DOCKER_NETWORK}

docker-apps: ## Build apps binaries for dockerized skywire-visor. `go build` with  ${DOCKER_OPTS}
	-${DOCKER_OPTS} go build -race -o ./node/apps/skychat.v1.0 ./cmd/apps/skychat
	-${DOCKER_OPTS} go build -race -o ./node/apps/helloworld.v1.0 ./cmd/apps/helloworld
	-${DOCKER_OPTS} go build -race -o ./node/apps/socksproxy.v1.0 ./cmd/apps/therealproxy
	-${DOCKER_OPTS} go build -race -o ./node/apps/socksproxy-client.v1.0  ./cmd/apps/therealproxy-client
	-${DOCKER_OPTS} go build -race -o ./node/apps/SSH.v1.0  ./cmd/apps/therealssh
	-${DOCKER_OPTS} go build -race -o ./node/apps/SSH-client.v1.0  ./cmd/apps/therealssh-client

docker-bin: ## Build `skywire-visor`, `skywire-cli`, `hypervisor`, `therealssh-cli`. `go build` with  ${DOCKER_OPTS}
	${DOCKER_OPTS} go build -race -o ./node/skywire-visor ./cmd/skywire-visor

docker-volume: dep docker-apps docker-bin bin  ## Prepare docker volume for dockerized skywire-visor
	-${DOCKER_OPTS} go build  -o ./docker/skywire-services/setup-node ./cmd/setup-node
	-./skywire-cli node gen-config -o  ./skywire-visor/skywire.json -r
	perl -pi -e 's/localhost//g' ./node/skywire.json # To make node accessible from outside with skywire-cli

docker-run: docker-clean docker-image docker-network docker-volume ## Run dockerized skywire-visor ${DOCKER_NODE} in image ${DOCKER_IMAGE} with network ${DOCKER_NETWORK}
	docker run -it -v $(shell pwd)/node:/sky --network=${DOCKER_NETWORK} \
		--name=${DOCKER_NODE} ${DOCKER_IMAGE} bash -c "cd /sky && ./skywire-visor skywire.json"

docker-setup-node:	## Runs setup-node in detached state in ${DOCKER_NETWORK}
	-docker container rm setup-node -f
	docker run -d --network=${DOCKER_NETWORK}  	\
	 				--name=setup-node	\
	 				--hostname=setup-node	skywire-services \
					  bash -c "./setup-node setup-node.json"

docker-stop: ## Stop running dockerized skywire-visor ${DOCKER_NODE}
	-docker container stop ${DOCKER_NODE}

docker-rerun: docker-stop
	-./skywire-cli gen-config -o ./node/skywire.json -r
	perl -pi -e 's/localhost//g' ./node/skywire.json # To make node accessible from outside with skywire-cli
	${DOCKER_OPTS} go build -race -o ./node/skywire-visor ./cmd/skywire-visor
	docker container start -i ${DOCKER_NODE}

run-syslog: ## Run syslog-ng in docker. Logs are mounted under /tmp/syslog
	-rm -rf /tmp/syslog
	-mkdir -p /tmp/syslog
	-docker container rm syslog-ng -f
	docker run -d -p 514:514/udp  -v /tmp/syslog:/var/log  --name syslog-ng balabit/syslog-ng:latest 


integration-startup: ## Starts up the required transports between `skywire-visor`s of interactive testing environment
	./integration/startup.sh

integration-teardown: ## Tears down all saved configs and states of integration executables
	./integration/tear-down.sh

integration-run-generic: ## Runs the generic interactive testing environment
	./integration/run-generic-env.sh

integration-run-messaging: ## Runs the messaging interactive testing environment
	./integration/run-messaging-env.sh

integration-run-proxy: ## Runs the proxy interactive testing environment
	./integration/run-proxy-env.sh

integration-run-ssh: ## Runs the ssh interactive testing environment
	./integration/run-ssh-env.sh

mod-comm: ## Comments the 'replace' rule in go.mod
	./ci_scripts/go_mod_replace.sh comment go.mod

mod-uncomm: ## Uncomments the 'replace' rule in go.mod
	./ci_scripts/go_mod_replace.sh uncomment go.mod

trigger-swc-build: ## Trigger integration build in skywire-services
	./ci_scripts/trigger-swc-build.sh

vendor-integration-check: ## Check compatibility of master@skywire-services with last vendored packages
	./ci_scripts/vendor-integration-check.sh



help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	
