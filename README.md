[![Build Status](https://travis-ci.com/watercompany/skywire.svg?token=QxVQj6gVZDzoFxD2YG65&branch=master)](https://travis-ci.com/watercompany/skywire)

# Skywire Mainnet - Public Test Phase

## Notes on this release

This is a public testing version of the Skywire mainnet and is intended for developers use to find bugs only. It is not yet intended to replace the testnet and miners should not install this software on their miners or they may lose their reward eligibility. 

The software is still under heavy development and the current version is intended for public testing purposes only. A GUI interface and various guides on how to use Skywire, application development on Skywire and contribution policies will follow in the near future. For now this version of the software can be used by developers to test the functionality and file bug issues to help the development. 

## Architecture 

Skywire is a decentralized and private network. Skywire separates the data and control plane of the network and assigns the tasks of network coordination and administration to dedicated services, while the nodes follow the rules that were created by the control plane and execute them. 

The core of Skywire is the Skywire node which hosts applications and is the gateway to use the network. It establishes connections, called transports, to other nodes, requests the setup of routes and forwards packets for other nodes on a route. The Skywire node exposes an API to applications for using the networking protocol of Skywire. 

In order to detach control plane tasks from the network nodes, there are 3 other services that maintain a picture of the network topology, calculate routes (currently based on the number of hops, but will be extended to other metrics) and set the routing rules on the nodes. 

The transport discovery maintains a picture of the network topology, by allowing Skywire nodes to advertise transports that they established with other nodes. It also allows to upload a status to indicate whether a given transport is currently working or not.

On the basis of this information the route finder calculates the most efficient route in the network. Nodes request a route to a given public key and the route finder will calculate the best route and return the transports that the packet will be sent over to reach the intended node. 

This information is sent from a node to the Setup Node, which sets the routing rules in all nodes along a route. Skywire nodes determine, which nodes they accept routing rules from, so only a whitelisted node can send routing rules to a node in the network. The only information the Skywire node gets for routing is a Routing ID and an associated rule that defines which transport to send a packet to (or to consume the packet). Therefore nodes along a route only know the last and next hop along the route, but not where the packet originates from and where it is sent to. Skywire supports source routing, so nodes can specify a path that a packet is supposed to take in the network. 

There are currently two types of transports that nodes can use. The messaging transport is a transport between two nodes that uses an intermediary messaging server to relay packets between them. The connection to a specific node and the connection to a messaging server is facilitated by a discovery service, that allows nodes to advertise the messaging servers over which they can be contacted. This transport is used by the setup node to send routing rules and can be used for other applications as well. It allows nodes behind NATs to communicate. The second transport type is TCP, which sets up a connection between two servers with a public IP. More transport types will be supported in the future and custom transport implementations can be written for specific use cases.

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

```bash
make # this will install all dependencies and build binaries
```

# Install skywire-node, skywire-cli, manager-node and therealssh-cli

```bash
make install
```

# Generate default json config

```bash
skywire-cli config
```

### Run `skywire-node`

`skywire-node` hosts apps, proxies app's requests to remote nodes and exposes communication API that apps can use to implement communication protocols. App binaries are spawned by the node, communication between node and app is performed via unix pipes provided on app startup.

```bash
# Run skywire-node. It takes one argument; the path of a configuration file (`skywire.json` if unspecified).
$ skywire-node skywire.json
```

### Run `skywire-node` in docker container

```bash
make node
```

### Run `skywire-cli`

The `skywire-cli` tool is used to control the `skywire-node`. Refer to the help menu for usage:

```bash
$ skywire-cli -h

# Command Line Interface for skywire
#
# Usage:
#   skywire-cli [command]
#
# Available Commands:
#   add-rule          adds a new routing rule
#   add-transport     adds a new transport
#   apps              lists apps running on the node
#   config            Generate default config file
#   find-routes       lists available routes between two nodes via route finder service
#   find-transport    finds and lists transport(s) of given transport ID or edge public key from transport discovery
#   help              Help about any command
#   list-rules        lists the local node's routing rules
#   list-transports   lists the available transports with optional filter flags
#   messaging         manage operations with messaging services
#   rm-rule           removes a routing rule via route ID key
#   rm-transport      removes transport with given id
#   rule              returns a routing rule via route ID key
#   set-app-autostart sets the autostart flag for an app of given name
#   start-app         starts an app of given name
#   stop-app          stops an app of given name
#   transport         returns summary of given transport by id
#   transport-types   lists transport types used by the local node
#
# Flags:
#   -h, --help         help for skywire-cli
#       --rpc string   RPC server address (default "localhost:3435")
#
# Use "skywire-cli [command] --help" for more information about a command.

```

### Apps

After `skywire-node` is up and running with default environment, default apps are run with the configuration specified in `skywire.json`. Refer to the following for usage of the default apps:

- [Chat](/cmd/apps/chat)
- [Hello World](/cmd/apps/helloworld)
- [The Real Proxy](/cmd/apps/therealproxy) ([Client](/cmd/apps/therealproxy-client))
- [The Real SSH](/cmd/apps/therealssh) ([Client](/cmd/apps/therealssh-client))

### Transports

In order for a local Skywire App to communicate with an App running on a remote Skywire node, a transport to that remote Skywire node needs to be established.

Transports can be established via the `skywire-cli`.

```bash
# Establish transport to `0276ad1c5e77d7945ad6343a3c36a8014f463653b3375b6e02ebeaa3a21d89e881`.
$ skywire-cli add-transport 0276ad1c5e77d7945ad6343a3c36a8014f463653b3375b6e02ebeaa3a21d89e881

# List established transports.
$ skywire-cli transports list
```

## App programming API

App is a generic binary that can be executed by the node. On app
startup node will open pair of unix pipes that will be used for
communication between app and node. `app` packages exposes
communication API over the pipe.

```golang
// Config defines configuration parameters for App
&app.Config{AppName: "helloworld", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
// Setup setups app using default pair of pipes
func Setup(config *Config) (*App, error) {}

// Accept awaits for incoming loop confirmation request from a Node and
// returns net.Conn for a received loop.
func (app *App) Accept() (net.Conn, error) {}

// Addr implements net.Addr for App connections.
&Addr{PubKey: pk, Port: 12}
// Dial sends create loop request to a Node and returns net.Conn for created loop.
func (app *App) Dial(raddr *Addr) (net.Conn, error) {}

// Close implements io.Closer for App.
func (app *App) Close() error {}
```

## Updater

This software comes with an updater, which is located in this repo: https://github.com/skycoin/skywire-updater. Follow the instructions in the README.md for further information. It can be used with a CLI for now and will be usable with the manager interface.
