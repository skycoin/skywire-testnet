.DEFAULT_GOAL := help
.PHONY : check lint install-linters dep test 
.PHONY : build  clean install  format  bin
.PHONY : host-apps bin 
.PHONY : run stop config
.PHONY : docker-image  docker-clean docker-network  
.PHONY : docker-apps docker-bin docker-volume 
.PHONY : docker-run docker-stop     

OPTS?=GO111MODULE=on 
DOCKER_IMAGE?=skywire-runner # docker image to use for running visor.`golang`, `buildpack-deps:stretch-scm`  is OK too
DOCKER_NETWORK?=SKYNET 
DOCKER_VISOR?=SKY01
DOCKER_OPTS?=GO111MODULE=on GOOS=linux # go options for compiling for docker container
TEST_OPTS?=-race -tags no_ci -cover -timeout=5m
BUILD_OPTS?=-race

check: lint test ## Run linters and tests

build: dep host-apps bin ## Install dependencies, build apps and binaries. `go build` with ${OPTS} 

run: stop build	config  ## Run visor on host
	./visor visor.json

stop: ## Stop running visor on host
	-bash -c "kill $$(ps aux |grep '[v]isor' |awk '{print $$2}')"

config: ## Generate visor.json
	-./skywire-cli visor gen-config -o  ./visor.json -r

clean: ## Clean project: remove created binaries and apps
	-rm -rf ./apps
	-rm -f ./visor ./skywire-cli ./setup-node ./hypervisor ./SSH-cli

install: ## Install `visor`, `skywire-cli`, `hypervisor`, `SSH-cli`
	${OPTS} go install ./cmd/visor ./cmd/skywire-cli ./cmd/setup-node ./cmd/hypervisor ./cmd/therealssh-cli

rerun: stop
	${OPTS} go build -race -o ./visor ./cmd/visor
	-./skywire-cli visor gen-config -o  ./visor.json -r
	perl -pi -e 's/localhost//g' ./visor.json
	./visor visor.json


lint: ## Run linters. Use make install-linters first	
	${OPTS} golangci-lint run -c .golangci.yml ./...
	# The govet version in golangci-lint is out of date and has spurious warnings, run it separately
	${OPTS} go vet -all ./...

vendorcheck:  ## Run vendorcheck
	GO111MODULE=off vendorcheck ./internal/... 
	GO111MODULE=off vendorcheck ./pkg/... 
	GO111MODULE=off vendorcheck ./cmd/apps/... 
	GO111MODULE=off vendorcheck ./cmd/hypervisor/...
	GO111MODULE=off vendorcheck ./cmd/setup-node/... 
	GO111MODULE=off vendorcheck ./cmd/skywire-cli/... 
	GO111MODULE=off vendorcheck ./cmd/visor/...
	# vendorcheck fails on ./cmd/therealssh-cli
	# the problem is indirect dependency to github.com/sirupsen/logrus
	#GO111MODULE=off vendorcheck ./cmd/therealssh-cli/... 	

test: ## Run tests
	-go clean -testcache &>/dev/null
	${OPTS} go test ${TEST_OPTS} ./internal/...
	#${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/...
	${OPTS} go test ${TEST_OPTS} ./pkg/app/...
	${OPTS} go test ${TEST_OPTS} ./pkg/cipher/...
	${OPTS} go test ${TEST_OPTS} ./pkg/dmsg/...
	${OPTS} go test ${TEST_OPTS} ./pkg/hypervisor/...
	${OPTS} go test ${TEST_OPTS} ./pkg/messaging-discovery/...
	${OPTS} go test ${TEST_OPTS} ./pkg/route-finder/...
	${OPTS} go test ${TEST_OPTS} ./pkg/router/...
	${OPTS} go test ${TEST_OPTS} ./pkg/routing/...
	${OPTS} go test ${TEST_OPTS} ./pkg/setup/...
	${OPTS} go test ${TEST_OPTS} ./pkg/transport/...
	${OPTS} go test ${TEST_OPTS} ./pkg/transport-discovery/...
	${OPTS} go test ${TEST_OPTS} ./pkg/visor/...
	${OPTS} go test  -tags no_ci -cover -timeout=5m ./pkg/messaging/...


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
bin: ## Build `hypervisor`, `skywire-cli`, `SSH-cli`, `visor`
	${OPTS} go build ${BUILD_OPTS} -o ./hypervisor ./cmd/hypervisor
	${OPTS} go build ${BUILD_OPTS} -o ./skywire-cli  ./cmd/skywire-cli
	${OPTS} go build ${BUILD_OPTS} -o ./setup-node ./cmd/setup-node
	${OPTS} go build ${BUILD_OPTS} -o ./messaging-server ./cmd/messaging-server
	${OPTS} go build ${BUILD_OPTS} -o ./SSH-cli ./cmd/therealssh-cli
	${OPTS} go build ${BUILD_OPTS} -o ./visor ./cmd/visor


release: ## Build `hypervisor`, `skywire-cli`, `SSH-cli`, `visor` and apps without -race flag
	${OPTS} go build -o ./hypervisor ./cmd/hypervisor
	${OPTS} go build -o ./skywire-cli  ./cmd/skywire-cli 
	${OPTS} go build -o ./setup-node ./cmd/setup-node
	${OPTS} go build -o ./SSH-cli ./cmd/therealssh-cli
	${OPTS} go build -o ./visor ./cmd/visor
	${OPTS} go build -o ./apps/skychat.v1.0 ./cmd/apps/skychat	
	${OPTS} go build -o ./apps/helloworld.v1.0 ./cmd/apps/helloworld
	${OPTS} go build -o ./apps/socksproxy.v1.0 ./cmd/apps/therealproxy
	${OPTS} go build -o ./apps/socksproxy-client.v1.0  ./cmd/apps/therealproxy-client
	${OPTS} go build -o ./apps/SSH.v1.0  ./cmd/apps/therealssh
	${OPTS} go build -o ./apps/SSH-client.v1.0  ./cmd/apps/therealssh-client

# Dockerized visor
docker-image: ## Build docker image `skywire-runner`
	docker image build --tag=skywire-runner --rm  - < skywire-runner.Dockerfile

docker-clean: ## Clean docker system: remove container ${DOCKER_VISOR} and network ${DOCKER_NETWORK}
	-docker network rm ${DOCKER_NETWORK} 
	-docker container rm --force ${DOCKER_VISOR}

docker-network: ## Create docker network ${DOCKER_NETWORK}
	-docker network create ${DOCKER_NETWORK}

docker-apps: ## Build apps binaries for dockerized visor. `go build` with  ${DOCKER_OPTS}
	-${DOCKER_OPTS} go build -race -o ./visor/apps/skychat.v1.0 ./cmd/apps/skychat
	-${DOCKER_OPTS} go build -race -o ./visor/apps/helloworld.v1.0 ./cmd/apps/helloworld
	-${DOCKER_OPTS} go build -race -o ./visor/apps/socksproxy.v1.0 ./cmd/apps/therealproxy
	-${DOCKER_OPTS} go build -race -o ./visor/apps/socksproxy-client.v1.0  ./cmd/apps/therealproxy-client
	-${DOCKER_OPTS} go build -race -o ./visor/apps/SSH.v1.0  ./cmd/apps/therealssh
	-${DOCKER_OPTS} go build -race -o ./visor/apps/SSH-client.v1.0  ./cmd/apps/therealssh-client

docker-bin: ## Build `visor`, `skywire-cli`, `visor`, `therealssh-cli`. `go build` with  ${DOCKER_OPTS}
	${DOCKER_OPTS} go build -race -o ./visor/visor ./cmd/visor

docker-volume: dep docker-apps docker-bin bin  ## Prepare docker volume for dockerized visor
	-${DOCKER_OPTS} go build  -o ./docker/skywire-services/setup-node ./cmd/setup-node
	-./skywire-cli visor gen-config -o  ./visor/visor.json -r
	perl -pi -e 's/localhost//g' ./visor/visor.json # To make visor accessible from outside with skywire-cli

docker-run: docker-clean docker-image docker-network docker-volume ## Run dockerized visor ${DOCKER_VISOR} in image ${DOCKER_IMAGE} with network ${DOCKER_NETWORK}
	docker run -it -v $(shell pwd)/visor:/sky --network=${DOCKER_NETWORK} \
		--name=${DOCKER_VISOR} ${DOCKER_IMAGE} bash -c "cd /sky && ./visor visor.json"

docker-setup-node:	## Runs setup-visor in detached state in ${DOCKER_NETWORK}
	-docker container rm setup-node -f
	docker run -d --network=${DOCKER_NETWORK}  	\
	 				--name=setup-node	\
	 				--hostname=setup-node	skywire-services \
					  bash -c "./setup-node setup-node.json"

docker-stop: ## Stop running dockerized visor ${DOCKER_VISOR}
	-docker container stop ${DOCKER_VISOR}

docker-rerun: docker-stop
	-./skywire-cli gen-config -o  ./visor/visor.json -r
	perl -pi -e 's/localhost//g' ./visor/visor.json # To make visor accessible from outside with skywire-cli
	${DOCKER_OPTS} go build -race -o ./visor/visor ./cmd/visor
	docker container start -i ${DOCKER_VISOR}

run-syslog: ## Run syslog-ng in docker. Logs are mounted under /tmp/syslog
	-rm -rf /tmp/syslog
	-mkdir -p /tmp/syslog
	-docker container rm syslog-ng -f
	docker run -d -p 514:514/udp  -v /tmp/syslog:/var/log  --name syslog-ng balabit/syslog-ng:latest 


integration-startup: ## Starts up the required transports between 'visor's of interactive testing environment
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

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	
