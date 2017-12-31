![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

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

### Windows

Right click on "Git Bash Here" in the folder
```
mkdir.exe -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone -b dev https://github.com/skycoin/skywire.git
```

Build the binaries for skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

Binaries will be built to $GOPATH/bin


## Run Skywire

### Unix systems
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Open a new command window

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999 -address :5000 -web-port :6001
```
### Windows

```
cd $GOPATH/bin
./manager.exe -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Open a new command window

```
cd $GOPATH/bin
./node.exe -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999 -address :5000 -web-port :6001
```
Use the browser to open http://127.0.0.1:8000

### Docker

```
docker build -t skycoin/skywire .
```

#### Start the manager

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```

Open [http://localhost:8000](http://localhost:8000).

#### Start a node and connect it to the manager

```
docker volume create skywire-data
docker run -ti --rm \
  --name=skywire-node \
  -v skywire-data:/root/.skywire \
  --link skywire-manager \
  -p 5000:5000 \
  -p 6001:6001 \
  skycoin/skywire \
    node \
      -connect-manager \
      -manager-address skywire-manager:5998 \
      -manager-web skywire-manager:8000 \
      -address :5000 \
      -web-port :6001
```
