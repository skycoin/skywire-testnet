# `dmsg`

## Terminology

- **entity -** A service of `dmsg` (typically being part of an executable running on a machine).
- **entity type -** The type of entity. `dmsg` has three entity types; `dmsg.Server`, `dmsg.Client`, `dmsg.Discovery`.
- **entry -** A data structure that describes an entity and is stored in `dmsg.Discovery` for entities to access.
- **frame -** The data unit of the `dmsg` system.
- **frame type -** The type of `dmsg` frame. There are four frame types; `REQUEST`, `ACCEPT`, `CLOSE`, `FWD`, `ACK`.
- **connection -** The direct line of duplex communication between a `dmsg.Server` and `dmsg.Client`.
- **transport -** A line of communication between two `dmsg.Client`s that is proxied via a `dmsg.Server`.
- **transport ID -** A uint16 value that identifies a transport.

## Frames

Frames have two sections; the header and the payload.

```
|| FrameType | TransportID | PayloadSize || Payload ||
|| 1 byte    | 2 bytes     | 2 bytes     || ~ bytes ||
```



## Entities

The `dmsg` system is made up of three entity types:
- `dmsg.Discovery` is a RESTful API that allows `dmsg.Client`s to find remote `dmg.Client`s and `dmsg.Server`s.
- `dmsg.Server` proxies frames between clients.
- `dmsg.Client` establishes transports between itself and remote `dmsg.Client`s.

### `dmsg.Discovery`

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


