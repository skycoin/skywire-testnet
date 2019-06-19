# `dmsg`

>**TODO:**
>
>- `ACK` frames should include the first 4 bytes of the rolling hash of incoming payloads, enforcing reliability of data. Transports should therefore keep track of incoming/outgoing rolling hashes.
>- Transports should also be noise-encrypted. `REQUEST` and `ACCEPT` frames should include noise handshake messages (`KK` handshake pattern), and the `FWD` and `ACK` payloads are to be encrypted.
> - `dmsg.Server` should check incoming frames to disallow excessive sending of `CLOSE`, `ACCEPT` and `REQUEST` frames.

## Terminology

- **entity -** A service of `dmsg` (typically being part of an executable running on a machine).
- **entity type -** The type of entity. `dmsg` has three entity types; `dmsg.Server`, `dmsg.Client`, `dmsg.Discovery`.
- **entry -** A data structure that describes an entity and is stored in `dmsg.Discovery` for entities to access.
- **frame -** The data unit of the `dmsg` system.
- **frame type -** The type of `dmsg` frame. There are four frame types; `REQUEST`, `ACCEPT`, `CLOSE`, `FWD`, `ACK`.
- **connection -** The direct line of duplex communication between a `dmsg.Server` and `dmsg.Client`.
- **transport -** A line of communication between two `dmsg.Client`s that is proxied via a `dmsg.Server`.
- **transport ID -** A uint16 value that identifies a transport.

## Entities

The `dmsg` system is made up of three entity types:
- `dmsg.Discovery` is a RESTful API that allows `dmsg.Client`s to find remote `dmg.Client`s and `dmsg.Server`s.
- `dmsg.Server` proxies frames between clients.
- `dmsg.Client` establishes transports between itself and remote `dmsg.Client`s.

Entities of types `dmsg.Server` or `dmsg.Client` are represented by a `secp256k1` public key.

```
           [D]

     S(1)        S(2)
   //   \\      //   \\
  //     \\    //     \\
 C(A)    C(B) C(C)    C(D)
```

Legend:
- ` [D]` - `dmsg.Discovery`
- `S(X)` - `dmsg.Server`
- `C(X)` - `dmsg.Client`

## Connection Handshake

A Connection refers to the line of communication between a `dmsg.Client` and `dmsg.Server`.

To set up a `dmsg` Connection, `dmsg.Client` dials a TCP connection to the `dmsg.Server` and then they perform a handshake via the [noise protocol](http://noiseprotocol.org/) using the `XK` handshake pattern (with the `dmsg.Client` as the initiator).

Note that `dmsg.Client` always initiates the `dmsg` connection, and it is a given that a `dmsg.Client` always knows the public key that identifies the `dmsg.Server` that it wishes to connect with.

## Frames

Frames are sent and received within a `dmsg` connection after the noise handshake. A frame has two sections; the header and the payload. Here are the fields of a frame:

```
|| FrameType | TransportID | PayloadSize || Payload ||
|| 1 byte    | 2 bytes     | 2 bytes     || ~ bytes ||
```

- The `FrameType` specifies the frame type via the one byte.
- The `TransportID` contains an encoded `uint16` which represents a identifier for a transport. A set of IDs are unique for a given `dmsg` connection.
- The `PayloadSize` contains an encoded `uint16` which represents the size (in bytes) of the payload.
- The `Payload` have a size that is obtainable via `PayloadSize`.

The following is a summary of the frame types:

| FrameType | Name | Payload Contents | Payload Size |
| --- | --- | --- | --- |
| `0x1` | `REQUEST` | initiating client's public key + responding client's public key | 66 |
| `0x2` | `ACCEPT` | initiating client's public key + responding client's public key | 66 |
| `0x3` | `CLOSE` | 1 byte that represents the reason for closing | 1 |
| `0xa` | `FWD` | uint16 sequence + transport payload | >2 |
| `0xb` | `ACK` | uint16 sequence | 2 |

## Transports

Transports are represented by transport IDs and facilitate duplex communication between two `dmsg.Client`s which are connected to a common `dmsg.Server`.

Transport IDs are assigned in such a manner:
- A `dmsg.Client` manages the assignment of even transport IDs between itself and each connected `dmsg.Server`. The set of transport IDs will be unique between itself and each `dmsg.Server`.
- A `dmsg.Server` manages the assignment of odd transport IDs between itself and each connected `dmsg.Client`. The set of transport IDs will be unique between itself and each `dmsg.Client`.

For a given transport:
- Between the initiating client and the common server - the transport ID is always a even value.
- Between the responding client and the common server - the transport ID is always a odd value.

Hence, a transport in it's entirety, is represented by 2 transport IDs.

### Transport Establishment

1. The initiating client chooses an even transport ID and forms a `REQUEST` frame with the chosen transport ID, initiating client's public key (itself) and also the responding client's public key. The `REQUEST` frame is then sent to the common server. The transport ID chosen must be unused between the initiating client and the server.
2. The common server receives the `REQUEST` frame and checks the contents. If valid, and the responding client exists, the server chooses an odd transport ID, swaps this original transport ID of the `REQUEST` frame with the chosen odd transport ID, and continues to forward it to the responding client. In doing this, the server records a rule relating the initiating/responding clients and the associated odd/even transport IDs.
3. The responding client receives the `REQUEST` frame and checks the contents. If valid, the responding client sends an `ACCEPT` frame (containing the same payload as the `REQUEST`) back to the common server. The common server changes the transport ID, and forwards the `ACCEPT` to the initiating client.

On any step, if an error occurs, any entity can send a `CLOSE` frame.

### Acknowledgement Logic

Each `FWD` frame is to be met with an `ACK` frame in order to be considered delivered.

- Each `FWD` payload has a 2-byte prefix (represented by a uint16 sequence). This sequence is unique per transport.
- The destination of the transport, after receiving the `FWD` frame, responds with an `ACK` frame with the same sequence as the payload.

## `dmsg.Discovery`

### Entry

An entry within the `dmsg.Discovery` can either represent a `dmsg.Server` or a `dmsg.Client`. The `dmsg.Discovery` is a key-value store, in which entries (of either server or client) use their public keys as their "key".

The following is the representation of an Entry in Golang.

```golang
// Entry represents an entity's entry in the Discovery database.
type Entry struct {
    // The data structure's version.
    Version string `json:"version"`

    // A Entry of a given public key may need to iterate. This is the iteration sequence.
    Sequence uint64 `json:"sequence"`

    // Timestamp of the current iteration.
    Timestamp int64 `json:"timestamp"`

    // Public key that represents the entity.
    Static cipher.PubKey `json:"static"`

    // Contains the entity's required client meta if it's to be advertised as a Client.
    Client *Client `json:"client,omitempty"`

    // Contains the entity's required server meta if it's to be advertised as a Server.
    Server *Server `json:"server,omitempty"`

    // Signature for proving authenticity of of the Entry.
    Signature cipher.Sig `json:"signature,omitempty"`
}

// Client contains the entity's required client meta, if it is to be advertised as a Client.
type Client struct {
    // DelegatedServers contains a list of delegated servers represented by their public keys.
    DelegatedServers []cipher.PubKey `json:"delegated_servers"`
}

// Server contains the entity's required server meta, if it is to be advertised as a Messaging Server.
type Server struct {
    // IPv4 or IPv6 public address of the Messaging Server.
    Address string `json:"address"`

    // Number of connections still available.
    AvailableConnections int `json:"available_connections"`
}
```

**Definition rules:**

- A record **MUST** have either a "Server" field, a "Client" field, or both "Server" and "Client" fields. In other words, a Messaging Node can be a Messaging Server Node, a Messaging Client Node, or both a Messaging Server Node and a Messaging Client Node.

**Iteration rules:**

- The first entry submitted of a given static public key, needs to have a "Sequence" value of `0`. Any future entries (of the same static public key) need to have a "Sequence" value of `{previous_sequence} + 1`.
- The "Timestamp" field of an entry, must be of a higher value than the "Timestamp" value of the previous entry.

**Signature Rules:**

The "Signature" field authenticates the entry. This is the process of generating a signature of the entry:

1. Obtain a JSON representation of the Entry, in which:
   1. There is no whitespace (no ` ` or `\n` characters).
   2. The `"signature"` field is non-existent.
2. Hash this JSON representation, ensuring the above rules.
3. Create a Signature of the hash using the node's static secret key.

The process of verifying an entry's signature will be similar.

### Endpoints

Only 3 endpoints need to be defined; Get Entry, Post Entry, and Get Available Servers.

#### GET Entry

Obtains a messaging node's entry.

> `GET {domain}/discovery/entries/{public_key}`

**REQUEST**

Header:

```
Accept: application/json
```

**RESPONSE**

Possible Status Codes:

- Success (200) - Successfully updated record.

  - Header:

    ```
    Content-Type: application/json
    ```

  - Body:

    > JSON-encoded entry.

- Not Found (404) - Entry of public key is not found.

- Unauthorized (401) - invalid signature.

- Internal Server Error (500) - something unexpected happened.

#### POST Entry

Posts an entry and replaces the current entry if valid.

> `POST {domain}/discovery/entries`

**REQUEST**

Header:

```
Content-Type: application/json
```

Body:

> JSON-encoded, signed Entry.

**RESPONSE**

Possible Response Codes:

- Success (200) - Successfully registered record.
- Unauthorized (401) - invalid signature.
- Internal Server Error (500) - something unexpected happened.

#### GET Available Servers

Obtains a subset of available server entries.

> `GET {domain}/discovery/available_servers`

**REQUEST**

Header:

```
Accept: application/json
```

**RESPONSE**

Possible Status Codes:

- Success (200) - Got results.

  - Header:

    ```
    Content-Type: application/json
    ```

  - Body:

    > JSON-encoded `[]Entry`.

- Not Found (404) - No results.

- Forbidden (403) - When access is forbidden.

- Internal Server Error (500) - Something unexpected happened.


