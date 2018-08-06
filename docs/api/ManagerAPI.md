v# Skywire Manager API Documentation
**Note: This document is a work in progress**

The following describes the Web API for the Skywire Manager (`manager`) application. You will need access to a running instance this application in order to utilise the APIs.

Examples provided below assume the Manager is running on the local machine (127.0.0.1). The default port for accessing the API is `8000`. 
All Node and Application keys have been deliberatly altered to ensure they are invalid.

## Manager API
The following API services are made avaiable by the Skywire Manager application (`manager`):
- [Manager](#manager)
    - [Login](#login)
    - [Check Login](#heck-login)
    - [Change Password](#update-password)
    - [Manager Node Request](#manager-node-request)
    - [Manager Term](#manager-term)
    - [Get Manager Port](#get-manager-port)
    - [Get Token](#get-token)
- [Connections](#run)
    - [Get All Connections](#get-all-connections)
    - [Get Manager Information](#get-manager-information)
    - [Get Node Information](#get-node-information)
    - [Set Node Configuration](#set-node-configuration)
    - [Get Node Configuration](#get-node-configuration)
    - [Save Client Connection](#save-client-connection)
    - [Remove Client Conneciton](#remove-client-connection)
    - [Edit Client Connection](#edit-client-connection)
    - [Get Client Connection](#get-client-connection)

### Login
Login (authenticate) to the Manager. This is the equivelant of logging into the Manager from the Web UI and is a pre-requisit for a number of other API calls.

Successfuly calling this API will setup an authenticated session with the Manager, and a session cookie will be provided back to the caller as part of the response payload. The session cookie is required as input to a number of other APIs.

The password must be provided as the value for the `pass` parameter.
The following validation is performed on the value provided for `pass`:
- Not less than 4 or larger than 20 characters. The call will return `false` if this condition is not met.
- Compares a hashed version of the provided password against the stored password hash. If they are not the same the service will return `false`.

#### Usage
```
URI: /login
Method: Post
```

Example:
```sh
curl -X "POST" "http://127.0.0.1:8000/login" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     --data-urlencode "pass=example-password"
```

Successful Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Set-Cookie: SWSId=1134c7bfcfa34d5c1015dfd473ab0cfa; Path=/; Expires=Wed, 18 Jul 2018 13:44:36 GMT; Max-Age=3600; HttpOnly
Date: Wed, 18 Jul 2018 12:44:36 GMT
Content-Length: 4
Connection: close

true
```

Error Response:
```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Date: Wed, 18 Jul 2018 12:42:22 GMT
Content-Length: 22
Connection: close

authentication failed
```

### Check Login
Check Login verifies that the current session is still authorised with the Manager. If an error response is returned, the client must Login again to re-authenticate.

#### Usage
```
URI: /checkLogin
Method: Post
```
Example:
```sh
## Manager - checkLogin
curl -X "POST" "http://127.0.0.1:8000/checkLogin" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     -H 'Cookie: SWSId=1134c7bfcfa34d5c1015dfd473ab0cfa'

```

Successful Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 20 Jul 2018 10:33:22 GMT
Content-Length: 32
Connection: close

6230fa3ce167dc9afcac12fe1a0125fc
```

Error Response:
```
HTTP/1.1 302 Found
Content-Type: text/plain; charset=utf-8
Set-Cookie: SWSId=1134c7bfcfa34d5c1015dfd473ab0cfa; Path=/; Expires=Fri, 20 Jul 2018 11:31:43 GMT; Max-Age=3600; HttpOnly
X-Content-Type-Options: nosniff
Date: Fri, 20 Jul 2018 10:31:43 GMT
Content-Length: 18
Connection: close

Unauthorized
false
```

### Update Password
Updates (changes) the Manager password.

Both the old and new passwords must be provided as values for the `oldPass` and `newPass` parameters.
The following validation is performed on both values, with errors returned of the validation fails:
- Not less than 4 or larger than 20 characters.
- The new password is not the system default password.
- The old (current) password is correct.
- The new password is successfully saved.

Note that the current authenticated session is destroyed when the password is changed. You will need to re-authenticate with the manager using the new password.s

#### Usage
```
URI: /updatePass
Method: Post
```

Example:
```sh
curl -X "POST" "http://127.0.0.1:8000/updatePass" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     --data-urlencode "oldPass=oldtest" \
     --data-urlencode "newPass=newtest"

```

Successful Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Set-Cookie: SWSId=; Path=/; Expires=Fri, 20 Jul 2018 11:01:26 GMT; Max-Age=0; HttpOnly
Date: Fri, 20 Jul 2018 11:01:26 GMT
Content-Length: 4
Connection: close

true
```

Error Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 20 Jul 2018 11:03:07 GMT
Content-Length: 28
Connection: close

New password length is 4~20.
```

### Manager Node Request
Asks the Manager to perform an API request for one of its connected Nodes. Use the `/conn/getNode` API to obtain the `IP` and `Port` for the Node you wish to call an API for.

You must have logged in to the Manager using `/login` and obtained a valid Token from `/token` before calling this API.

Set the following parameters on the request:
- `addr` = URI of the Node API to be called
- `method` = The HTTP method to be used (GET, POST, etc)


#### Usage
```
URI: /req
Method: Post
```
Example - Requesting `/node/getInfo`:
```sh
curl -X "POST" "http://127.0.0.1:8000/req" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     -H 'Cookie: SWSId=15723926ac7a3b0d4659cb472ed3cab2' \
     --data-urlencode "addr=http://{IP:PORT}/node/getInfo" \
     --data-urlencode "method=post"
```

Response:
```json
{"discoveries":{"discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68":true},"transports":null,"app_feedbacks":null,"version":"0.1.0","tag":"dev","os":"linux"}
```

Example - Requesting `/node/getApps`:
```sh
curl -X "POST" "http://127.0.0.1:8888/req" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     -H 'Cookie: SWSId=15723926ac7a3b0d4659cb472ed3cab2' \
     --data-urlencode "addr=http://{IP:PORT}/node/getApps" \
     --data-urlencode "method=post"
```

Response:
```json
[{"key":"123e56823dc4b83472648172af3936618bca3663a52429fc32afc1870937830496","attributes":["sockss"],"allow_nodes":null}]
```

### Manager Term
#### Usage
```
URI: /term
Method: Post
```
Example:
```sh
```

Response:
```json
```

### Get Manager Port
Retreives the Port number that is used by the Manager. The default is `8000`.

#### Usage
```
URI: /getPort
Method: Get
```
Example:
```sh
curl "http://127.0.0.1:8000/getPort" \
     -H 'Authorization: ' \
     -H 'Cookie: SWSId=bdfbdcda4981101816c1e4b57de1e5de'
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 20 Jul 2018 11:28:22 GMT
Content-Length: 4
Connection: close

8000
```

### Get Token
Get an API authentication token from the Manager. This is a pre-requisit for calling a number of APIs.

The `token` returned in the body of the response (as per the example below) is required as a parameter for other API calls.

#### Usage

```
URI: /getToken
Method: Get
```

Example Request:
```sh
curl "http://127.0.0.1:8000/getToken"
```

Example Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 18 Jul 2018 11:53:12 GMT
Content-Length: 64
Connection: close

bf43103c60b1eb30f8cacd619f0b4c7c8feacd6e0fc40ff1c6d3d3573c1d6fd7
```

## Connections
### Get All Connections
Get all currently active Node connections from the Manager. There are currently no pre-requisits for calling this API (do not need to be logged in or authenticated).

If the request is successful a JSON array containing zero, one or more Node connections (`Conn struct`) will be returned. This represents the list of Nodes currently connected to the Manager.

The `Conn struct` contains the following elements:
* Key (`string`) - The Node key.
* Type (`string`) - The connection type. Can be either `TCP` or `UDP`.
* SendBytes (`unit64`) - The number of bytes that have been sent to this node.
* RecvBytes (`unit64`) - The number of bytes that have been received by this node.
* LastActTime (`int64`) - The last time the Manager recieved an Acknowledgement from the Node.
* StartTime (`int64`) - The time the Node was started.

#### Usage

```
URI: /conn/getAll
Method: Get
```

Example Request:
```sh
curl "http://127.0.0.1:8000/conn/getAll"
```

Successful Response:
```json
[
  {
    "key": "01baac57c217b77c70c2c71b78e2445f14a9fc6397341eaab23fec62c6bac42f1c",
    "type": "TCP",
    "send_bytes": 1001,
    "recv_bytes": 1127,
    "last_ack_time": 55,
    "start_time": 4615
  }

  {
    "key": "01bcaf37c253b77c70c2c74a78e2445f14a9ff6397341eaab01fec62c3bfc41a1c",
    "type": "TCP",
    "send_bytes": 100,
    "recv_bytes": 115,
    "last_ack_time": 35,
    "start_time": 4011
  }
]
```

Error Response:
```
HTTP/1.1 500 Internal Server Error
```

### Get Manager Information
Get information about the Manager. There are currently no pre-requisits for calling this API (do not need to be logged in or authenticated).

If the request is successful the result provided in the response body is a `string` composed of the following elements in the following format:
```
[Host]:[Port]-[Manager Public Key]
```

#### Usage
```
URI: /getServerInfo
Method: Get
```

Example:
```sh
curl "http://127.0.0.1:8000/conn/getServerInfo"
```

Successful Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 18 Jul 2018 12:14:35 GMT
Content-Length: 81
Connection: close

127.0.0.1:5998-110510c20ac843f63fb1d001b5c1e23b9b46ce6efb9a2c57379f8a67b976af0c11
```

Error Response:
```
HTTP/1.1 500 Internal Server Error
```

### Get Node Information
Get detailed information from the Manager about the specified Node. The Node key must be passed as a query string to the URI.

Use the `/conn/getAll` to obtain a list of the connect Node keys, then use one of the returned Node Keys as input to this API request.

Note: You must have already logged into the Manager using the `/login` API, and obtained a valid token from the `/token` API.

#### Usage
```
URI: /conn/getNode?key={node=key}
Method: Post
```
Example:
```sh
curl -X "POST" "http://127.0.0.1:8000/conn/getNode?key=06bfac57c217b77c70c2c71b78e2445f14a9fc6397341eaab23fec62c6bac42c1c" \
     -H 'Cookie: SWSId=aeecf9e41466555a16f9a5891e084eb6'
```

Response:
```json
{"type":"TCP","addr":"127.0.0.1:6001","send_bytes":322,"recv_bytes":455,"last_ack_time":22,"start_time":82}
```

### Set Node Configuration
#### Usage
```
URI: /conn/setNodeConfig
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### Get Node Configuration
#### Usage
```
URI: /conn/getNodeConfig
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### Save Client Connection
#### Usage
```
URI: /conn/saveClientConnection
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### Remove Client Connections
#### Usage
```
URI: /conn/removeClientConnections
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### Edit Client Connection
#### Usage
```
URI: /conn/editClientConnection
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### GetClientConnection
#### Usage
```
URI: /conn/getClientConnection
Method: TBA
```
Example:
```sh
```

Response:
```json
```
