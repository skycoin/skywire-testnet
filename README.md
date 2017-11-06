![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

Skywire
=======

All the configuration is kept in `/etc/meshnet.cfg` (example in `meshnet-example.cfg`)

To run socks server with, for example, 2 hops in meshnet:

```sh
go run cmd/demo/socks/socks.go 2
```

To run vpn proxy server with, for example, 2 hops in meshnet:

```sh
go run cmd/demo/vpn/vpn.go 2
```


# Running skywire

## Run server

```sh
go run cmd/rpc/server/rpc-server.go
```

It will run the rpc server to accept messages on `localhost` on port which
environment variable `MESH_RPC_PORT` is assigned to.
If no such variable, it will work on port `1234`.

## Run client

```sh
go run cmd/rpc/cli/rpc-cli.go
```

It will run rpc client which will send message to port `1234`.

If you want another port to send messages, point it as an argument like this:

```sh
go run cmd/rpc/cli/rpc-cli.go 2222 # will send requests to port 2222
```

## Open client web interface in browser

To run client in a browser interface run `cmd/rpc/cli/rpc-cli.sh` which will open web interface on port 9999,
so you can use it in your browser like http://the-url-which-the-client-is-situated-at:9999.
This way needs [gotty](https://github.com/yudai/gotty) to be installed.

### Install gotty on linux

```sh
go get github.com/yudai/gotty
```

### Install gotty on macOS

```sh
brew tap yudai/gotty
brew install gotty
```

# Dependencies

Dependencies are managed with [dep](https://github.com/golang/dep).

To install `dep`:

```sh
go get -u github.com/golang/dep
```

`dep` vendors all dependencies into the repo.

If you change the dependencies, you should update them as needed with `dep ensure`.

Use `dep help` for instructions on vendoring a specific version of a dependency, or updating them.

After adding a new dependency (with `dep ensure`), run `dep prune` to remove any unnecessary subpackages from the dependency.

When updating or initializing, `dep` will find the latest version of a dependency that will compile.

## dep examples

### Initialize all dependencies

```sh
dep init
dep prune
```

### Update all dependencies

```sh
dep ensure -update -v
dep prune
```

### Add a single dependency (latest version)

```sh
dep ensure github.com/foo/bar
dep prune
```

### Add a single dependency (more specific version), or downgrade an existing dependency

```sh
dep ensure github.com/foo/bar@tag
dep prune
```

# Short guide

## Nodes

Nodes can be connected to each other directly by Transports (one transport per paired node) or through a sequence of nodes by Connections (one Connection per paired node)

They are connected to nodemanager through controlConn, to apps atteched to them throguh AppTalkAddr and to its transports.

When node is creating it sends register request to nodemanager which address was pointed as a parameter; nodemanager gives a node an id (pubkey), bufferSize and other params which are common to all nodes in the meshnet, and starts to listen for connection messages.

In order to connect directly with other node Node receives from nodemanager TransportCreateCM message and creates a transport (at the same time the node to pair with also creates a transport symmetrically) - setTransportFromMessage method.


Routing is applied by using RouteRule struct

RouteRule fields are:

IncomingRoute - route by which next RouteRule will be picked from node's routeForwardingRules (OutgoingRoute of the previous RouteRule)
IncomingTransport - transport from which messages is come (a pair of the OutgoingTransport of the previous RouteRule); zero means the beginning of the route
OutgoingRoute - a route, which will be used as IncomingRoute at the next node; zero means the end of the route
OutgoingTransport - transport to which a message will be resent; zero means the end of the route

Node adds route rules by receiving a message from nodemanager (AddRouteCM) with all route info. Then it adds it to routeForwardingRule map.

After receiving InRouteMessage from a transport node looks into the routing info of the message, makes sure that incoming route and transport exist and then depending on if OutgoingTransport and OutgoingRoute exist it repacks it into OutRouteMessage with new In and Out route and transport (taken from its own routeForwardingRules table) and sends this message to the transport. If there are no OutgoingTransport and OutgoingRoute in the InRouteMessage, node supposes the message datagram to be a connection message, extracts the connection id and sends this message contents to that of his connections selected by this id.


Connections are created by node when it wants to connect with a node which is not connected with it directly, by Dial method. Node sends a request to establish a connection to nodemanager which finds the shortest route between two nodes and sends to them messages to assign connections (message types involved: AssignConnectionCM, AddRouteCM, ConnectionOnCM). Connections contain the first routeId as a starting point for node to decide where to send a message from connection. Connections accept messages from apps, split them if needed and resend to node as ConnectionMessage, then node sends them through meshnet. At the other side other part of connection receives connection messages from the node, assembles them and sends to the receiver app.


Node waits for app messages through appTalkPort. Application can send register request to node and in this way it creates a conn with it and send register request further to nodemanager. App knows only node address and nothing more about meshnet. In order to work app can send requests to connect with other apps, sends messages to them and gets lists of another apps in the meshnet. Node handles all types of these messages and sends results back to app, also it receives messages from another apps and nodemanager and resends it back to app through appConn (node_app.go)


## Transports

transports are for pairing nodes.

Each node has a transport per every node it is paired with. Transport are created by transport factory from nodemanager. When nodemanager gets a command to connect two nodes directly it creates transport factory which creates two transports by sending messages to nodes through control channel. Then both nodes create transports to connect with each other (by CreateTransportFromMessage method).

Transports can receive OutRouteMessage from nodes attached to them through a channel (incomingFromNode) and TransportDatagramTransfer from a transport paired to it through UDP. Messages for sending to a paired transport are collected in pendingOut queue and then are sent to the paired transport by UDP. In the case of fail tries continue to repeat until retransmitLimit is reached.

When transport receives a message through UDP (by receiveFromPair()) it looks if it is TransportDatagramTransfer or ack. In the first case it sends ack to the pair, repacks TransportDatagramTransfer to InRouteMessage and sends it to then node which looks what to do with that further (acceptAndSendAck method). In the second case it notices a routine which sent a message that ack is come (through ackChannel).

If sending is failed transport goes to "disconnected" state.

TransportInfo is a brief form of Transport for using only by rpc client.


## Nodemanager

Nodemanager makes following:

register nodes(by accepting RegisterNodeCM messages from nodes)
connect nodes directly(by creating transport factories and sending TransportCreateCM and OpenUDPCM messages to nodes)
connect them with routes(by finding optimal route between them and send AddRouteCM messages to nodes)
receive register requests from apps through nodes and pass them to apptracker(RegisterAppCM messages)
launching and shutting down transports(Shutdown)
talking with apptracker (and other services then) in order to pass requests/responses to/from it


Register nodes:

Nodemanager accepts RegisterNodeCM message and if Connect field is TRUE, it connects the node directly with any random node in the meshnet (for testing purposes, can be avoided in the future). After that it forms a new Id for the node and saves NodeRecord which is a brief version of Node.

NodeRecord is necessary to Nodemanager to have info about how all nodes connected in order to find routes between them(NodeRecord.getTransportToNode is used by route finding service)

Node ID and some network info like MaxBuffer, MaxPacketSize, TimeUnit, SendInterval, ConnectionTimeout which are common for all nodes in the meshnet, are sent to the node in the reply message (RegisterNodeCMAck)


Connecting nodes directly

Nodemanager connects nodes directly after receiving ConnectDirectlyCM mesage from a node.
It creates TransportFactory with two TransportRecord entities(brief copy of paired transports)

after that it assigns UDP ports to nodes which they will be use for connecting with each other (one port for one paired node, through OpenUDPCM message) and give them command to create transports (TransportCreateCM).


Connecting with route (will be moved in the future from here to RouteServers)

Nodes that can't be connected directly can be connected by routes
Node sends ConnectWithRouteCM to Nodemanager; Nodemanager uses its routeGraph to find optimal path. routeGraph is formed by rebuildRoutes method using the info kept in NodeRecords for bulding graph. After the graph is build, nodemanager can use it by applying Dijkstra algorithm (so far all weights are 1, it will be changed in the future by applying info from Logistics server). It makes RouteRule for each node completes the route and sends AddRouteCM messages to them.
After that it sends SetConnectionOn messages to both nodes which are intended to connect and connections change their staus to CONNECTED


## Applications

Apps are used for end-users, e.g. socks or vpn servers, they use meshnet to connect with each other. App needs to register at node which it can use for connection with meshnet (RegisterAtNodeCM message). It should know only its ip host and connects to it by tcp.

In order to be created app needs to point node ip address:port as a param. Then it sends register request to node (RegisterAtNode method) and after that can send to it requests about list of another apps in the meshnet (GetAppsList method), connecting to another app (Connect method), sending messages to it.
App talks with node by sending/receiving first 8 bytes which contain the size of the message and then the message itself (sendToNode and getFullMessage methods)

App has an id and type fields in text form which are pointed by user. Type is needed that app can be found by another apps (say, "socks_server" etc.) by using AppTracker

App receives messages from end-user through an address pointed as ProxyAddress field. It is a host which enduser should point in it's user application, e.g.browser.

App should contain consume(msg *messages.AppMessage) method in order to know what to do with AppMessage messages received from meshnet.


Examples of apps are in the folder

LocalClient and LocalServer are for tests, they make no sense in practical use.

different implementations of app can redetermine its methods due to their features (Send, consume etc)


## AppTracker

AppTracker serves like storage for all apps registered in the meshnet. Nodemanager can't run without it.

Apptracker receives info about new service from nodemanager (which receives it from app itself through the node attached to app) and keeps it, then it gave it back in the case of nodemanager request (typically other apps require this info to see what apps they can paired with in order to run, say socks client app searches for socks server apps etc.)

Messages involved: AppRegistrationRequest for app registration and AppListRequest for returning the list of existing apps.


## Misc

### Acks handling works this way:

Sender has sequence field which increments an every message sending. This sequence is assigned to the message as a field value.
Before sending the message response channel is created and assigned to responseChannels map by key=sequence.
Then sender sends the message and waits when something will go to this channel.
When message has reached its receiver, receiver sends back an ack with the same sequence value as it received.
Then ack turns back to the sender side, the method which receives messages on the sender side looks in it, extracts the sequence value, selects a channel from the map of responseChannels by the sequence and sends "true" to the Channel.
Sender gets it to the channel, unblocks it's execution and ensures that ack has come.
The channel can be deleted as this sequence value will never be repeated again.
That resolves the situation when acks returned back not in the order in which messages has gone to the receiver.
