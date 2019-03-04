# skywire-node
Implementation of a Skywire Node.

A instance is configured through the next configuration structure:

```go
type Config struct {
	Client      bool // Whether this node is to act as a Messaging Client.
	Server      bool // Whether this node is to act as a Messaging Server.
	Public      bool // Whether this node is to be advertised in Discovery.
	CommandLine bool // Whether this node allows command-line interactions via RPC.

	DiscoveryAddresses []string // Addresses of the Discovery nodes in which the Messaging Node is to use.
	ListenRPC          string   // RPC Listening Address for command-line.
	ListenTCP          string   // (Server only) TCP Listening address of this Messaging Node.

	StaticPubKey cipher.PubKey // StaticID public key which identifies the node.
	StaticSecKey cipher.SecKey // StaticID secret key which identifies the node.

	DelegatedServers []cipher.PubKey // (Client only) Messaging Servers to connect to first.
	ServerCount      int             // (Client only) Number of Messaging Servers to connect to initially.
}
```

A instance contains the following fields:

```go
	config            *Config
	clients           *InterestedClients
	pool              *net.Pool
	entry             *client.Entry
	discoveryClient   client.APIClient
	associatedEntries map[cipher.PubKey]*client.Entry
	linker            *keyLinker
	deliveryPool      *deliveryPool
```

Where:

1. `config` is the previously mentioned configuration.
2. `clients` is a set of Interested Clients.
Interested clients are:
Clients that this instance has attempted to send a message to.
Clients that had previously sent an acknowledge message to this instance.
3. `pool` holds a pool of connections, we use the method `pool.Get`
every time we need to retrieve a connection using either a public key
or an ephemeral public key.
4. `entry` holds a messaging-discovery entry that represents the current instance. An
initial entry is created by calling `createInitialEntry` function during
instance initialization. Every time that a client instance connects to new nodes
or drops connections to other nodes the entry must be updated with the
new list of ephemeral keys. After that the entry in messaging-discovery should
also be updated.
5. `discoveryClient` holds a client instance connecting to a messaging-discovery
instance, through it we can Set, Update or Recover entries from
messaging-discovery.
6. `associatedEntries` holds a map that allows the instance to recover a
previously retrieved from messaging-discovery entry given the public key of a
instance. Entries that are not related anymore should be deleted.
7. `linker` is a struct that allows to recover the static key associated
with a given ephemeral key. It has the methods `StaticOfEphemeral`,
`SetLink` and `RemoveLink`.
8. `deliveryPool` is a struct that allows to keep track of which servers
we have tried to rely a message through when trying to send a segment
to a given client (indexed by their static public key). Is used for
retry logic purposes, also keep tracks of the segment ID that should be
used in the message to be delivered to said client and holds data that
allows to recreate the message we are trying to deliver using a
different server as a rely. Has the methods:
`Delete`, `GetLastWorkingEphemeral`, `TryGetNewEphemeral` and
`GetDestinationData`.

# Code structure

#### Files
- `api_client`: contains client code for communication with the RPC
server.
- `api_gateway`: contains an implementation of an RPC API gateway, it
uses the data types contained in rpc directory. Uses grpc and protocol
buffers, though it also exposes an interface for the Gateway to allow
different implementations.
- `delivery_pool`: contains delivery pool implementation.
- `delivery_pool_test`: contains delivery_pool unit tests.
- `handlers`: contains the instance handlers, those are fired when a given
message type is received or upon specific events:
```go
	callbacks := &node.Callbacks{
		BasicForwardResponseClient: func(_ *node.Node, _ *net.Conn, srcPort, dstPort uint16) {
		    // code. Fired when a basic forward response is received by a client node
		},
		BasicForward: func(node *Node, conn *net.Conn, sender cipher.PubKey, srcPort, dstPort uint16, payload []byte) (ack bool) {
		    // code. Fired when a basic forward segment is received by a client node
		},
	    BasicForwardResponseServer: func(node *Node, conn *net.Conn) {
	        // code. Fired when a basic forward response is received by a server node
	    },
	    Close: func(node *Node, conn *net.Conn) {
	        // code. Fired when a connection is closed
	    },
	}
```
We can pass these callbacks to the instance on startup with `instance.Start(callbacks)`
- `handlers_test`: contains handlers unit tests.
- `header`: contains implementations for the [header](#header) and its [flags](#flags).
- `header_test`: contains header and flags unit tests.
- `interested_clients`: interested clients implementation. Currently
not used.
- `instance`: instance's core code as well as linker implementation.
- `node_start`: startup code for client and server instance.
- `node_test`: contains unit tests for the instance, currently tests the
correct creation of a messaging-discovery entry that represents the instance.
- `segment`: contains [basic forward](#basic-forward) and [basic forward response](#basic-forward-response) segments
implementation.
- `segment_test`: contains segments unit tests.
- `text_cipher`: contains "PlainText" implementation, which allows
encoding/decoding for plain text that is embedded in the segment.
- `text_cipher_test`: contains text cipher unit tests.

# Installation
Vendoring directory is provided, but dependencies can be rebuild with [dep](https://golang.github.io/dep/docs/installation.html) by running the command `dep ensure`

To build the binaries:
```bash
go install ./pkg/...
```
After that you should have a binary called messaging-discovery-service in your `$GOPATH`

# Usage

**command line applications**

There are 3 command line applications related to instance: `messaging-client`,
`simple-server` and `client-cli`.

`simple-server`: Starts a instance server, which needs to connect to a
skywire messaging-discovery server. The server will rely messages between clients.

    -a value, --address value    tcp address of the client to listen (default: "localhost:8080")
    -d value, --discovery value  discovery address which the server needs to connect (default: "http://localhost:9090")
    --help, -h                   show help
    --version, -v                print the version

`messaging-client`: Starts a instance client server. The client needs to
connect to a messaging-discovery server plus at least one server instance.

       -a value, --address value  tcp address of the client to listen (default: "localhost:8080")
       --discovery value          discovery address which the server needs to connect (default: "http://localhost:9090")
       --help, -h                 show help
       --version, -v              print the version

`client-cli`: Is a command line utility to interact with instance clients,
so far it allows to send messages from one client to another relied by
server nodes.

Right now client-cli has only a sub-command available: `send-message`,
which accept the following options:

       -p value, --public-key value      public key of the target node that we are sending the message to
       -c value, --client-address value  tcp address of the client node that exposes an rpc server (default: "localhost:8080")
       -d value, --dst-port value        destination node port (default: 8080)
       -s value, --src-port value        source node port (default: 8080)

**usage example**

First of all, assuming that you have compiled the binaries and they
are available through your $PATH, we can setup at least one instance server.

As a prerequisite we are assuming that there is a messaging-discovery instance running
in "localhost:9090".

```bash
simple-server -a "localhost:8080" -d "localhost:9090"
```
You should be able to see something like that after running the command,
(the instance will initialize with a different public key):

```bash
[2018-10-12T16:19:54+08:00] INFO [app]: server started with public key: 02a409e5652290009e5a37aea1dc3b4c99c871228c8d0032a96d3515c44a684c3b
[2018-10-12T16:19:54+08:00] INFO [node]: server started. Listening in localhost:8080
```

Now we can setup a couple client nodes:

```bash
simple-client --discovery "localhost:9090" -a "localhost:8081"
```

As an output of this command you should be able to see something like
that:

```bash
[2018-10-12T16:28:22+08:00] INFO [app]: server started with public key: 03b4dea1197c62f988d7ac8f29c35d4753a6f6d867491ada3c6143eecb9b01684e
[2018-10-12T16:28:22+08:00] DEBUG [node]: current pool (ephemeral key pairs):
[2018-10-12T16:28:22+08:00] DEBUG [node]: my ephemeral: 032a5915284d9c0994e06cbf7e09d613c817b8f79a857f75f4ebcb2a16a810d118
[2018-10-12T16:28:22+08:00] DEBUG [node]: remote ephemeral: 03e6e9a99b70cacf2bc58f7b8304b221418e1139fdb86fe0ae8fc74909d1abc4e4
[2018-10-12T16:28:22+08:00] INFO [node]: server started. Listening in localhost:8081
```

Then we start a second instance:

```bash
simple-client --discovery "localhost:9090" -a "localhost:8082"
```

Output:

```bash
[2018-10-12T16:41:46+08:00] INFO [app]: server started with public key: 0292e3af45a5aafb523c4033bb3f3d6736a87a2ae1b05a5406f4618931ec5a8cee
[2018-10-12T16:41:46+08:00] DEBUG [node]: current pool (ephemeral key pairs):
[2018-10-12T16:41:46+08:00] DEBUG [node]: my ephemeral: 025719dd14f9e571c22ec387754bb06a8f7a1eb58e7cda0131ed0d62642fd4e0be
[2018-10-12T16:41:46+08:00] DEBUG [node]: remote ephemeral: 03fd3b146910e8bf99cfd4cd991fe4281bd46b584b4893860eb68be1c9ce41b899
[2018-10-12T16:41:46+08:00] INFO [node]: server started. Listening in localhost:8082
```

Now we can effectively send messages from one client instance to another by
using the client-cli to tell one instance which message to deliver to a
destination instance, we will tell which destination instance by using the
destination client public key:

```bash
client-cli send-message -c "localhost:8081" -p "0292e3af45a5aafb523c4033bb3f3d6736a87a2ae1b05a5406f4618931ec5a8cee" "Hi, I'm node 1 :)"
```

With this command we are telling client on localhost:8081 to say
"Hi, I'm instance 1 :)" to the client instance which public key is
"0292e3af45a5aafb523c4033bb3f3d6736a87a2ae1b05a5406f4618931ec5a8cee".

If everything goes well we will see something like this:
```bash
[2018-10-12T16:49:46+08:00] INFO [app]: message delivered
```

Then if we check the logs for the client instance listening in
"localhost:8081" we can see these logs:

```bash
[2018-10-12T16:49:19+08:00] INFO [node]: entry recovered from discovery has ephemeral keys: [{025719dd14f9e571c22ec387754bb06a8f7a1eb58e7cda0131ed0d62642fd4e0be 02a409e5652290009e5a37aea1dc3b4c99c871228c8d0032a96d3515c44a684c3b}]
[2018-10-12T16:49:19+08:00] INFO [node]: found a server to rely with pk: 02a409e5652290009e5a37aea1dc3b4c99c871228c8d0032a96d3515c44a684c3b
[2018-10-12T16:49:19+08:00] DEBUG [node]: ciphered message with shared secret product of these keys:
[2018-10-12T16:49:19+08:00] DEBUG [node]: remote public ephemeral: 025719dd14f9e571c22ec387754bb06a8f7a1eb58e7cda0131ed0d62642fd4e0be
[2018-10-12T16:49:19+08:00] DEBUG [node]: local private ephemeral: f0e08a346d54e54b8210b58e0ba1ab42afec0fdc74015010ff83bc4f5adf56a4
[2018-10-12T16:49:19+08:00] INFO [node]: sending payload which header receiver is: 025719dd14f9e571c22ec387754bb06a8f7a1eb58e7cda0131ed0d62642fd4e0be
[2018-10-12T16:49:19+08:00] DEBUG [node]: ClientOnBasicForwardResponse called
[2018-10-12T16:49:19+08:00] INFO [node]: segment acknowledged by receiver
[2018-10-12T16:49:19+08:00] DEBUG [node]: ClientOnBasicForwardResponse processed
```

On the other hand if we look at the client instance at "localhost:8082" we
will be able to see these logs:

```bash
[2018-10-12T16:49:19+08:00] DEBUG [node]: ciphered message with shared secret product of these keys:
[2018-10-12T16:49:19+08:00] DEBUG [node]: remote public ephemeral: 032a5915284d9c0994e06cbf7e09d613c817b8f79a857f75f4ebcb2a16a810d118
[2018-10-12T16:49:19+08:00] DEBUG [node]: local private ephemeral: 121459f0b1f4c0016bbb743b39dab0554e640dc1a7a571840260a854b673214e
[2018-10-12T16:49:19+08:00] INFO [node]: received segment number 0 with message: Hi, I'm node 1 :)
```

And finally if we look at the instance server logs we can see how it relied
the message to the destination client and the response to the caller:

```bash
[2018-10-12T16:49:19+08:00] DEBUG [node]: ServerOnBasicForward called
[2018-10-12T16:49:19+08:00] DEBUG [node]: BasicForwardAction relied
[2018-10-12T16:49:19+08:00] DEBUG [node]: ServerOnBasicForwardResponse called
[2018-10-12T16:49:19+08:00] DEBUG [node]: BasicForwardResponseAction relied
```

### Data structures
<a name="basic-forward"></a>Basic Forward:

| Field Name | Size (in bytes) | Description |
| ---------- | --------------- | ----------- |
| Header | 75 | Includes Sender and Receiver ephemeral keys and a segment ID. |
| CipherText | ~ | `chacha20poly1305` AEAD-Encrypted. Uses the Header field as AD, and the segment ID as the nonce. Data should have a 64 byte prefix that has Sender and Receiver static keys and ports. |

<a name="basic-forward-response"></a>Basic Forward Response:

| Field Name | Size (in bytes) | Description |
| ---------- | --------------- | ----------- |
| Flags | 1 | Contain the flags for the segment. |
| Sender | 33 | Ephemeral public key of a Sender. |
| Receiver | 33 | Ephemeral public key of a Receiver. |
| Segment ID | 8 | An big-endian encoded uint64 integer that uniquely identifies a segment. |

<a name="header"></a>Header:

| Field Name | Size (in bytes) | Description |
| ---------- | --------------- | ----------- |
| Sender Static | 33 | The static public key of the Sender. |
| Receiver Static | 33 | The static public key of the Receiver. |
| Sender Port | 2 | uint16 port of the Sender. |
| Receiver Port | 2 | uint16 port of the Receiver. |
| Payload | ~ | The actual payload to be delivered. |

<a name="flags"></a>Flags:
The "Flags" field has 8 possible bits to use. The following are defined:
- bit 0 & 1 (`RESP_CODE`) :
    - `00` - Unspecified Response / Rejected by Local Messaging Client.
    - `01` - Accepted by Remote Messaging Client.
    - `10` - Rejected by Messaging Server.
    - `11` - Rejected by Remote Messaging Server.