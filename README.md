![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# Skywire

Other languages:

* [中文文档](README-CN.md)
* [Spanish Document](README-ES.md)
* [Korean Document](README-KO.md)

Links:

* [Skywire node map](https://skycoin.github.io/skywire/)
* [Skywire Blog](https://blog.skycoin.net/tags/skywire/)

Skywire is still under heavy development.


![2018-01-21 10 44 06](https://user-images.githubusercontent.com/1639632/35190261-1ce870e6-fe98-11e7-8018-05f3c10f699a.png)

## Table of Contents
* [Requirements](#requirements)
* [Install](#install)
* [Run Skywire](#run-skywire)
* [Docker](#docker)
* [System Images Download Url](#images)
* Developers Guide
  * [Manager API](docs/api/ManagerAPI.md)
  * [Node API](docs/api/NodeAPI.md)

## Requirements

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)  https://github.com/golang/go/wiki/SettingGOPATH in our case GOPATH must point to ```/usr/local/skywire/go/```

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

#### Run Skywire Manager

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/manager_start
```

`tip: the manager start script will also run a local node, you don't need to run in manually on the manager.`

#### Run Skywire Node

Open a new command window on a node only computer

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/node_start
```

`tip: the node is instructed to connect to the manager IP automatically, if you use a non default IP set you must check the file "/etc/default/skywire" and change the MANAGER_IP variable on each Pc of your setup.`

This two files are the default start script for skywire services, take a peek on them to know more if yu are interested.

#### Stop Skywire Manager and Node.

If you started the manager and the nodes by the ways stated above you can stop them on each Pc by this command on a console:

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/stop
```

This will check for the pid of the running processes and kill them. If you ran them by hand using a call to a the specific manager or node binaries this will not stop them, in this case you must run this:

```
killall node
killall manager
```

##### Installing the manager and node as a service using systemd

If you use a modern Linux OS (released after 2017) you are using systemd as init manager, skywire has the files needed to make them a service inside systemd.

Please note that the manager instance will start also a local node, so you must select just a manager on a net and the rest will be nodes.

###### Installing mananger unit on systemd 

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-manager.service /etc/systemd/system/
systemctl enable skywire-manager.service
systemctl start skywire-manager.service
```

###### Installing node unit on systemd 

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-node.service /etc/systemd/system/
systemctl enable skywire-node.service
systemctl start skywire-node.service
```

## Open Skywire Manager View
Open [http://localhost:8000](http://localhost:8000).
The default login password for Skywire manager is **1234**.

### Connect to node
1) Connect to node —— Search services —— Connect

2) Connect to node —— Enter the key for node and app —— Connect

In the first way, you can search for nodes around the world, and select the nodes you want to connect to; The second way is to connect to the specified node.

#### Use Skywire App
After the default normal start, the App will display "** available port **" (e.g. 9443) after successful connection.

#### Use Firefox Browser

#### Install FoxyProxy Standard
Open Firefox Browser,address bar input"https://addons.mozilla.org/zh-CN/firefox/addon/foxyproxy-standard/", Click "add to Firefox" button to follow the prompts to install.

#### Configuration FoxyProxy Standard
After the installation is complete, browse the Firefox address bar enter about: "addons" into the plugin page, find FoxyProxy "Standard" and click on the preferences into the configuration page < br > select "Use Enabled Proxies By Patterns and Priority" enable FoxyProxy < br >
Click "Add" to Add the configuration,
```
Proxy Type: SOCKS5
IP address, DNS name, server name: 127.0.0.1
Port: 9443
```
And then finally click "Save"

### SSH tool

#### SSH
After this service is opened, the application public key will be generated. Based on the public key of the node and the public key, the node can be managed remotely in any machine running Skywire.

`Note: do not open SSH at will, and show the Node Key and App Key to strangers.`

#### SSH Client
Enter Node Key and App Key. After the connection is successful, the Port (Port) will be displayed under the button, for example, 30001, and finally, use any SSH remote connection tool connection.

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

**Note:**
The images of skywire for ARM v5 and v7 are built upon `busybox` whereas the ARM v6 and v8 containers will run on `alpine`.

Open [http://localhost:8000](http://localhost:8000).
The default login password for Skywire manager is **1234**.

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
      -web-port :6001 \
      -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68
```

### Docker Compose

```
docker-compose up
```

Open [http://localhost:8000](http://localhost:8000).

## Download System Images

<a name="images"></a>

Note: these images can only be run on [Orange Pi Prime](http://www.orangepi.cn/OrangePiPrime/index_cn.html).

### IP presetted system images

Default password is 'samos'.

### Upgrade the presetted system images

The base images has a few [known bugs](https://github.com/skycoin/skywire/issues/171), we have built a one time upgrade script to fix that until we upgrade the new presseted system images. 

If you want to upgrade the presetted system images please see [this one time upgrade instructions](static/script/upgrade/).

### Important:

Manager system image package contains Skywire Manager and a Skywire Node, other Node system image package only launch a Node.

1) Download [Manager](https://downloads3.skycoin.net/skywire-images/manager.tar.gz) (IP:192.168.0.2)

2) Download [Node1](https://downloads3.skycoin.net/skywire-images/node-1-03.tar.gz) (IP:192.168.0.3)

3) Download [Node2](https://downloads3.skycoin.net/skywire-images/node-2-04.tar.gz) (IP:192.168.0.4)

4) Download [Node3](https://downloads3.skycoin.net/skywire-images/node-3-05.tar.gz) (IP:192.168.0.5)

5) Download [Node4](https://downloads3.skycoin.net/skywire-images/node-4-06.tar.gz) (IP:192.168.0.6)

6) Download [Node5](https://downloads3.skycoin.net/skywire-images/node-5-07.tar.gz) (IP:192.168.0.7)

7) Download [Node6](https://downloads3.skycoin.net/skywire-images/node-6-08.tar.gz) (IP:192.168.0.8)

8) Download [Node7](https://downloads3.skycoin.net/skywire-images/node-7-09.tar.gz) (IP:192.168.0.9)

### Manually set IP system image

`Note: This system image only contains the basic environment of Skywire, and it needs to set IP, etc.`

Download [Pure Image](https://downloads3.skycoin.net/skywire-images/skywire_pure.tar.gz)

## Building the Orange Pi images yourself

The images are in https://github.com/skycoin/Orange-Pi-H5

Instructions for building the images are in https://github.com/skycoin/Orange-Pi-H5/wiki/How-to-build-the-images
