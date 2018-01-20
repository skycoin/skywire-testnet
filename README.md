![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# [中文文档](https://github.com/vyloy/skywire/blob/master/README-CN.md)
# Skywire

Here is our [Blog ](https://blog.skycoin.net/tags/skywire/) about Skywire.

Skywire is still under heavy development. 

![9461512959241_ pic](https://user-images.githubusercontent.com/1639632/33813339-fdcefb4e-de5d-11e7-867b-06b7d3f79be2.jpg)

### Requirements

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH
## Install 
### Unix systems

```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

Build the binaries for skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

## Run Skywire

### Unix systems
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Open a new command window

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999-028667f86c17f1b4120c5bf1e58f276cbc1110a60e80b7dc8bf291c6bec9970e74 -address :5000 -web-port :6001
```
Use the browser to open http://127.0.0.1:8000

