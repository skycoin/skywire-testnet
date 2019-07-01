[![Build Status](https://travis-ci.com/watercompany/skywire.svg?token=QxVQj6gVZDzoFxD2YG65&branch=master)](https://travis-ci.com/watercompany/skywire)

# Skywire Mainnet - Public Test Phase

- [Skywire Mainnet - Public Test Phase](#Skywire-Mainnet---Public-Test-Phase)
  - [Notes on this release](#Notes-on-this-release)
  - [Architecture](#Architecture)
  - [Build and run](#Build-and-run)
    - [Requirements](#Requirements)
    - [Build](#Build)
    - [Run `visor`](#Run-visor)
    - [Run `visor` in docker container](#Run-visor-in-docker-container)
    - [Run `skywire-cli`](#Run-skywire-cli)
    - [Apps](#Apps)
    - [Transports](#Transports)
  - [App programming API](#App-programming-API)
  - [Testing](#Testing)
    - [Testing with default settings](#Testing-with-default-settings)
    - [Customization with environment variables](#Customization-with-environment-variables)
      - [$TEST_OPTS](#TESTOPTS)
      - [$TEST_LOGGING_LEVEL](#TESTLOGGINGLEVEL)
      - [$SYSLOG_OPTS](#SYSLOGOPTS)
  - [Updater](#Updater)
  - [Running skywire in docker containers](#Running-skywire-in-docker-containers)
    - [Run dockerized `visor`](#Run-dockerized-visor)
      - [Structure of `./visor`](#Structure-of-visor)
    - [Refresh and restart `SKY01`](#Refresh-and-restart-SKY01)
    - [Customization of dockers](#Customization-of-dockers)
      - [1. DOCKER_IMAGE](#1-DOCKER_IMAGE)
      - [2. DOCKER_NETWORK](#2-DOCKER_NETWORK)
      - [3. DOCKER_VISOR](#3-DOCKER_NODE)
      - [4. DOCKER_OPTS](#4-DOCKER_OPTS)
    - [Dockerized `visor` recipes](#Dockerized-visor-recipes)
      - [1. Get Public Key of docker-visor](#1-Get-Public-Key-of-docker-visor)
      - [2. Get an IP of visor](#2-Get-an-IP-of-visor)
      - [3. Open in browser containerized `skychat` application](#3-Open-in-browser-containerized-skychat-application)
      - [4. Create new dockerized `visors`](#4-Create-new-dockerized-visors)
      - [5. Env-vars for develoment-/testing- purposes](#5-Env-vars-for-develoment-testing--purposes)
      - [6. "Hello-Mike-Hello-Joe" test](#6-%22Hello-Mike-Hello-Joe%22-test)

## Notes on this release

This is a public testing version of the Skywire mainnet and is intended for developers use to find bugs only. It is not yet intended to replace the testnet and miners should not install this software on their miners or they may lose their reward eligibility. 

The software is still under heavy development and the current version is intended for public testing purposes only. A GUI interface and various guides on how to use Skywire, application development on Skywire and contribution policies will follow in the near future. For now this version of the software can be used by developers to test the functionality and file bug issues to help the development. 

## Architecture 

Skywire is a decentralized and private network. Skywire separates the data and control plane of the network and assigns the tasks of network coordination and administration to dedicated services, while the visors follow the rules that were created by the control plane and execute them. 

The core of Skywire is the Skywire visor which hosts applications and is the gateway to use the network. It establishes connections, called transports, to other visors, requests the setup of routes and forwards packets for other visors on a route. The Skywire visor exposes an API to applications for using the networking protocol of Skywire. 

In order to detach control plane tasks from the network visors, there are 3 other services that maintain a picture of the network topology, calculate routes (currently based on the number of hops, but will be extended to other metrics) and set the routing rules on the visors. 

The transport discovery maintains a picture of the network topology, by allowing Skywire visors to advertise transports that they established with other visors. It also allows to upload a status to indicate whether a given transport is currently working or not.

On the basis of this information the route finder calculates the most efficient route in the network. Visors request a route to a given public key and the route finder will calculate the best route and return the transports that the packet will be sent over to reach the intended visor. 

This information is sent from a visor to the Setup Node, which sets the routing rules in all visors along a route. Skywire visors determine, which visors they accept routing rules from, so only a whitelisted visor can send routing rules to a visor in the network. The only information the Skywire visor gets for routing is a Routing ID and an associated rule that defines which transport to send a packet to (or to consume the packet). Therefore visors along a route only know the last and next hop along the route, but not where the packet originates from and where it is sent to. Skywire supports source routing, so visors can specify a path that a packet is supposed to take in the network. 

There are currently two types of transports that visors can use. The messaging transport is a transport between two visors that uses an intermediary messaging server to relay packets between them. The connection to a specific visor and the connection to a messaging server is facilitated by a discovery service, that allows visors to advertise the messaging servers over which they can be contacted. This transport is used by the setup visor to send routing rules and can be used for other applications as well. It allows visors behind NATs to communicate. The second transport type is TCP, which sets up a connection between two servers with a public IP. More transport types will be supported in the future and custom transport implementations can be written for specific use cases.

## Build and run

### Requirements

Skywire requires a version of [golang](https://golang.org/) with [go modules](https://github.com/golang/go/wiki/Modules) support.

### Build

```bash
# Clone.
$ git clone https://github.com/skycoin/skywire
$ cd skywire
$ git checkout mainnet
# Build
$ make build # installs all dependencies, build binaries and apps
```

**Note: Environment variable OPTS**

Build can be customized with environment variable `OPTS` with default value `GO111MODULE=on`

E.g.

```bash
$ export OPTS="GO111MODULE=on GOOS=darwin"
$ make
# or
$ OPTS="GSO111MODULE=on GOOS=linux GOARCH=arm" make
```

**Install visor, skywire-cli, hypervisor and SSH-cli**

```bash
$ make install  # compiles and installs all binaries
```

**Generate default json config**

```bash
$ skywire-cli visor gen-config
```

### Run `visor`

`visor` hosts apps, proxies app's requests to remote visors and exposes communication API that apps can use to implement communication protocols. App binaries are spawned by the visor, communication between visor and app is performed via unix pipes provided on app startup.

```bash
visor
$ visor visor-config.json
```

### Run `visor` in docker container

```bash
make docker-run
```

### Run `skywire-cli`

The `skywire-cli` tool is used to control the `visor`. Refer to the help menu for usage:

```bash
$ skywire-cli -h

# Command Line Interface for skywire
#
# Usage:
#   skywire-cli [command]
#
# Available Commands:
#   help        Help about any command
#   mdisc       Contains sub-commands that interact with a remote Messaging Discovery
#   visor       Contains sub-commands that interact with the local Visor
#   rtfind      Queries the Route Finder for available routes between two visors
#   tpdisc      Queries the Transport Discovery to find transport(s) of given transport ID or edge public key
#
# Flags:
#   -h, --help   help for skywire-cli
#
# Use "skywire-cli [command] --help" for more information about a command.

```

### Apps

After `visor` is up and running with default environment, default apps are run with the configuration specified in `visor-config.json`. Refer to the following for usage of the default apps:

- [Chat](/cmd/apps/skychat)
- [Hello World](/cmd/apps/helloworld)
- [The Real Proxy](/cmd/apps/therealproxy) ([Client](/cmd/apps/therealproxy-client))
- [The Real SSH](/cmd/apps/SSH) ([Client](/cmd/apps/SSH-client))

### Transports

In order for a local Skywire App to communicate with an App running on a remote Skywire visor, a transport to that remote Skywire visor needs to be established.

Transports can be established via the `skywire-cli`.

```bash
# Establish transport to `0276ad1c5e77d7945ad6343a3c36a8014f463653b3375b6e02ebeaa3a21d89e881`.
$ skywire-cli visor add-tp 0276ad1c5e77d7945ad6343a3c36a8014f463653b3375b6e02ebeaa3a21d89e881

# List established transports.
$ skywire-cli visor ls-tp
```

## App programming API

App is a generic binary that can be executed by the visor. On app
startup visor will open pair of unix pipes that will be used for
communication between app and visor. `app` packages exposes
communication API over the pipe.

```golang
// Config defines configuration parameters for App
&app.Config{AppName: "helloworld", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
// Setup setups app using default pair of pipes
func Setup(config *Config) (*App, error) {}

// Accept awaits for incoming loop confirmation request from a Visor and
// returns net.Conn for a received loop.
func (app *App) Accept() (net.Conn, error) {}

// Addr implements net.Addr for App connections.
&Addr{PubKey: pk, Port: 12}
// Dial sends create loop request to a Visor and returns net.Conn for created loop.
func (app *App) Dial(raddr *Addr) (net.Conn, error) {}

// Close implements io.Closer for App.
func (app *App) Close() error {}
```

## Testing

### Testing with default settings

```bash
$ make test
```

### Customization with environment variables

#### $TEST_OPTS

Options for `go test` could be customized with $TEST_OPTS variable

E.g.
```bash
$ export TEST_OPTS="-race -tags no_ci -timeout 90s -v"
$ make test
```

#### $TEST_LOGGING_LEVEL

By default all log messages during tests are disabled.
In case of need to turn on log messages it could be achieved by setting $TEST_LOGGING_LEVEL variable

Possible values:
- "debug"
- "info", "notice"
- "warn", "warning"
- "error"
- "fatal", "critical"
- "panic"

E.g.
```bash 
$ export TEST_LOGGING_LEVEL="info"
$ go clean -testcache || go test ./pkg/transport -v -run ExampleManager_CreateTransport
$ unset TEST_LOGGING_LEVEL
$ go clean -testcache || go test ./pkg/transport -v
```

#### $SYSLOG_OPTS

In case of need to collect logs in syslog during integration tests $SYSLOG_OPTS variable can be used.

E.g.
```bash
$ make run_syslog ## run syslog-ng in docker container with logs mounted to /tmp/syslog
$ export SYSLOG_OPTS='--syslog localhost:514'
$ make integration-run-messaging ## or other integration-run-* goal
$ sudo cat /tmp/syslog/messages ## collected logs from VisorA, VisorB, VisorC instances
```

## Updater

This software comes with an updater, which is located in this repo: https://github.com/skycoin/skywire-updater. Follow the instructions in the README.md for further information. It can be used with a CLI for now and will be usable with the manager interface.

## Running skywire in docker containers

There are two make goals for running in development environment dockerized `visor`.

### Run dockerized `visor`

```bash
$ make docker-run
```

This will:

- create docker image `skywire-runner` for running `visor`
- create docker network `SKYNET` (can be customized)
- create docker volume ./visor with linux binaries and apps
- create container  `SKY01` and starts it (can be customized)

#### Structure of `./visor`

```
./visor
├── apps                            # visor `apps` compiled with DOCKER_OPTS
│   ├── skychat.v1.0                   #
│   ├── helloworld.v1.0             #
│   ├── socksproxy-client.v1.0    #
│   ├── socksproxy.v1.0           #
│   ├── SSH-client.v1.0      #
│   └── SSH.v1.0             #
├── local                           # **Created inside docker**
│   ├── skychat                        #  according to "local_path" in visor-config.json
│   ├── socksproxy                #
│   └── SSH                  #
├── PK                              # contains public key of visor
├── skywire                         # db & logs. **Created inside docker**
│   ├── routing.db                  #
│   └── transport_logs              #
├── visor-config.json                    # config of visor
└── visor                    # `visor` binary compiled with DOCKER_OPTS
```

Directory `./visor` is mounted as docker volume for `visor` container.

Inside docker container it is mounted on `/sky`

Structure of `./visor` partially replicates structure of project root directory.

Note that files created inside docker container has ownership `root:root`, 
so in case you want to `rm -rf ./visor` (or other file operations) - you will need `sudo` it.

Look at "Recipes: Creating new dockerized visor" for further details.

### Refresh and restart `SKY01`

```bash
$ make refresh-visor
```

This will:

 - stops running visor
 - recompiles `visor` for container
 - start visor again

### Customization of dockers

#### 1. DOCKER_IMAGE

Docker image for running `visor`.

Default value: `skywire-runner` (built with `make docker-image`)

Other images can be used.
E.g.

```bash
DOCKER_IMAGE=golang make docker-run #buildpack-deps:stretch-scm is OK too
```

#### 2. DOCKER_NETWORK

Name of virtual network for `visor`

Default value: SKYNET

#### 3. DOCKER_VISOR

Name of container for `visor`

Default value: SKY01

#### 4. DOCKER_OPTS

`go build` options for binaries and apps in container.

Default value: "GO111MODULE=on GOOS=linux"

### Dockerized `visor` recipes

#### 1. Get Public Key of docker-visor

```bash
$ cat ./visor/visor.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' '
# 029be6fa68c13e9222553035cc1636d98fb36a888aa569d9ce8aa58caa2c651b45
```

#### 2. Get an IP of visor

```bash
$ docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' SKY01
# 192.168.112
```

#### 3. Open in browser containerized `skychat` application

```bash
$ firefox http://$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' SKY01):8000  
```

#### 4. Create new dockerized `visors`

In case you need more dockerized visors or maybe it's needed to customize visor
let's look how to create new visor.

```bash
# 1. We need a folder for docker volume
$ mkdir /tmp/SKYVISOR
visor
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/visor ./cmd/visor
# 3. compile apps
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/apps/skychat.v1.0 ./cmd/apps/skychat
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/apps/helloworld.v1.0 ./cmd/apps/helloworld
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/apps/socksproxy.v1.0 ./cmd/apps/therealproxy
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/apps/SSH.v1.0  ./cmd/apps/SSH
$ GO111MODULE=on GOOS=linux go build -o /tmp/SKYVISOR/apps/SSH-client.v1.0  ./cmd/apps/SSH-client
visor
$ skywire-cli visor gen-config -o /tmp/SKYVISOR/visor-config.json
# 2019/03/15 16:43:49 Done!
$ tree /tmp/SKYVISOR
# /tmp/SKYVISOR
# ├── apps
# │   ├── skychat.v1.0
# │   ├── helloworld.v1.0
# │   ├── socksproxy.v1.0
# │   ├── SSH-client.v1.0
# │   └── SSH.v1.0
# ├── visor-config.json
visor
# So far so good. We prepared docker volume. Now we can:
$ docker run -it -v /tmp/SKYVISOR:/sky --network=SKYNET --name=SKYVISOR skywire-runner bash -c "visor"
# [2019-03-15T13:55:08Z] INFO [messenger]: Opened new link with the server # 02a49bc0aa1b5b78f638e9189be4ed095bac5d6839c828465a8350f80ac07629c0
# [2019-03-15T13:55:08Z] INFO [messenger]: Updating discovery entry
# [2019-03-15T13:55:10Z] INFO [skywire]: Connected to messaging servers
# [2019-03-15T13:55:10Z] INFO [skywire]: Starting skychat.v1.0
# [2019-03-15T13:55:10Z] INFO [skywire]: Starting RPC interface on 127.0.0.1:3435
# [2019-03-15T13:55:10Z] INFO [skywire]: Starting socksproxy.v1.0
# [2019-03-15T13:55:10Z] INFO [skywire]: Starting SSH.v1.0
# [2019-03-15T13:55:10Z] INFO [skywire]: Starting packet router
# [2019-03-15T13:55:10Z] INFO [router]: Starting router
# [2019-03-15T13:55:10Z] INFO [trmanager]: Starting transport manager
# [2019-03-15T13:55:10Z] INFO [router]: Got new App request with type Init: {"app-name":"skychat",# "app-version":"1.0","protocol-version":"0.0.1"}
# [2019-03-15T13:55:10Z] INFO [router]: Handshaked new connection with the app skychat.v1.0
# [2019-03-15T13:55:10Z] INFO [skychat.v1.0]: 2019/03/15 13:55:10 Serving HTTP on :8000
# [2019-03-15T13:55:10Z] INFO [router]: Got new App request with type Init: {"app-name":"SSH",# "app-version":"1.0","protocol-version":"0.0.1"}
# [2019-03-15T13:55:10Z] INFO [router]: Handshaked new connection with the app SSH.v1.0
# [2019-03-15T13:55:10Z] INFO [router]: Got new App request with type Init: {"app-name":"socksproxy",# "app-version":"1.0","protocol-version":"0.0.1"}
# [2019-03-15T13:55:10Z] INFO [router]: Handshaked new connection with the app socksproxy.v1.0
```

Note that in this example docker is running in non-detached mode - it could be useful in some scenarios.

Instead of skywire-runner you can use:

- `golang`, `buildpack-deps:stretch-scm` "as is"
- and `debian`, `ubuntu` - after `apt-get install ca-certificates` in them. Look in `skywire-runner.Dockerfile` for example

#### 5. Env-vars for develoment-/testing- purposes

```bash
export SW_VISOR_A=127.0.0.1
export SW_VISOR_A_PK=$(cat ./visor-config.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' ')
export SW_VISOR_B=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' SKY01)
export SW_VISOR_B_PK=$(cat ./visor/visor-config.json|grep static_public_key |cut -d ':' -f2 |tr -d '"'','' ')
```

#### 6. "Hello-Mike-Hello-Joe" test

Idea of test from Erlang classics: https://youtu.be/uKfKtXYLG78?t=120

```bash
# Setup: run visors on host and in docker
$ make run
$ make docker-run
# Open in browser skychat application
$ firefox http://$SW_VISOR_B:8000  &
# add transport
$ ./skywire-cli add-transport $SW_VISOR_B_PK
# "Hello Mike!" - "Hello Joe!" - "System is working!"
$ curl --data  {'"recipient":"'$SW_VISOR_A_PK'", "message":"Hello Mike!"}' -X POST  http://$SW_VISOR_B:8000/message
$ curl --data  {'"recipient":"'$SW_VISOR_B_PK'", "message":"Hello Joe!"}' -X POST  http://$SW_VISOR_A:8000/message
$ curl --data  {'"recipient":"'$SW_VISOR_A_PK'", "message":"System is working!"}' -X POST  http://$SW_VISOR_B:8000/message
# Teardown
$ make stop && make docker-stop
```
