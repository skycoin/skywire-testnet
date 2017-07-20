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
