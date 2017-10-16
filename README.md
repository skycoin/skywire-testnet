### Requirements

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)

### Install

```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone -b dev https://github.com/skycoin/skywire.git
go get ./...
```
Build the web static files for monitor

Please read the README.md under the `web` folder before if have any questions

```
cd $GOPATH/src/github.com/skycoin/net/skycoin-messenger/monitor/web
./build.sh
```
Build the binaries for skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

Binaries will be built to $GOPATH/bin

### Run after boot
* $GOPATH/bin/manager on every skywire

  manager manages skywire nodes and provide a website for skywire user to control nodes.

  arguments:

```
Usage of ./manager:
  -address string
    	address to listen on (default ":5998")
  -web-dir string
    	monitor web page (default "/go/src/github.com/skycoin/net/skycoin-messenger/monitor/web/dist")
  -web-port string
    	monitor web page port (default ":8000")
```

```
./manager 
```

* $GOPATH/bin/node on every pi in skywire

  node transports the internet traffic for apps

  arguments:

```
Usage of ./node:
  -address string
    	address to listen on (default ":5000")
  -connect-manager
    	connect to manager if true
  -discovery-address value
    	addresses of discovery
  -manager-address string
    	address of node manager (default ":5998")
  -seed
    	use fixed seed to connect if true (default true)
  -seed-path string
    	path to save seed info (default "/root/.skywire/node/keys.json")
```

```
./node -connect-manager -manager-address 192.168.1.1:5998 -discovery-address messenger.skycoin.net:5999
```

