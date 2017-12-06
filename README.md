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
git clone -b dev https://github.com/skycoin/skywire.git
```

Build the binaries for skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

### Windows

Right click on "Git Bash Here" in the folder
```
mkdir.exe -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

Build the binaries for skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

Binaries will be built to $GOPATH/bin


## Run SkyWire

### Unix systems
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Open a new command window

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address www.yiqishare.com:5999 -address :5000 -web-port :6001 
```
### Windows

```
cd $GOPATH/bin
./manager.exe -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Open a new command window

```
cd $GOPATH/bin
./node.exe -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address www.yiqishare.com:5999 -address :5000 -web-port :6001 
```
Use the browser to open http://127.0.0.1:8000



