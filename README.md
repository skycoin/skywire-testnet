![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# [中文文档](https://github.com/vyloy/skywire/blob/master/README-CN.md)
# Skywire

Here is our [Blog ](https://blog.skycoin.net/tags/skywire/) about Skywire.

Skywire is still under heavy development. 
Skywire还在努力开发当中，如果没有太多技术背景，请等待年后发布的版本。



![2018-01-21 10 44 06](https://user-images.githubusercontent.com/1639632/35190261-1ce870e6-fe98-11e7-8018-05f3c10f699a.png)

## Table of Contents
* [Requirements](#requirements)
* [Install](#install)
* [Run Skywire](#run-skywire)
* [Docker](#run-skywire)

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

## Docker

```
docker build -t skycoin/skywire .
```

### Start the manager

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```

Open [http://localhost:8000](http://localhost:8000).

### Start a node and connect it to the manager

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

### Docker Compose

```
docker-compose up
```

Open [http://localhost:8000](http://localhost:8000).