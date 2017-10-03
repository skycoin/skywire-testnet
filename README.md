# Skywire

### How to test

#### Discovery

```
cd $GOPATH/src/github.com/skycoin/skywire/cmd/discovery
go run main.go
```

#### Node 1

```
cd $GOPATH/src/github.com/skycoin/skywire/cmd/node
go run main.go -manager-address :5999 -address :5000
```

#### Node 2

```
cd $GOPATH/src/github.com/skycoin/skywire/cmd/node
go run main.go -manager-address :5999 -address :5001
```

#### Socks5 Server

```
cd $GOPATH/src/github.com/skycoin/skywire/cmd/socks/server
go run server.go remote.go
```

#### Socks5 Client

```
cd $GOPATH/src/github.com/skycoin/skywire/cmd/socks/client
go run client.go local.go
```
