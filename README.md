![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# [中文文档](README-CN.md)
# [Spanish Document](README-ES.md)
# [Korean Document](README-KO.md)
# Skywire

Here is our [Blog ](https://blog.skycoin.net/tags/skywire/) about Skywire.

Skywire is still under heavy development.



![2018-01-21 10 44 06](https://user-images.githubusercontent.com/1639632/35190261-1ce870e6-fe98-11e7-8018-05f3c10f699a.png)

## Table of Contents
* [Requirements](#requirements)
* [Install](#install)
* [Run Skywire](#run-skywire)
* [Docker](#docker)
* [System Images Download Url](#images)

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

#### Run Skywire Manager
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

`tip: If you run with the above command, you will not be able to close the current window or you will close Skywire Manger.`

If you need to close the current window and continue to run Skywire Manager, you can use
```
cd $GOPATH/bin
nohup ./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager > /dev/null 2>&1 &sleep 3
```

`Note: do not execute the above two commands at the same time, just select one of them.`

#### Run Skywire Node

Open a new command window

```
cd $GOPATH/bin
./node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001
```

`tip: If you run with the above command, you will not be able to close the current window or you will close Skywire Node.`

If you need to close the current window and continue to run Skywire Manager, you can use
```
cd $GOPATH/bin
nohup ./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001 > /dev/null 2>&1 &cd /
```

#### Stop Skywire Manager and Node.

1) If the Skywire Manager and Node are started by using the terminal window, please press Ctrl + c on the respective terminal of Manager and Node.

2) Use the shutdown terminal to keep running, please enter:
##### Stop Skywire Manager
```
cd $GOPATH/bin
pkill -F manager.pid
```

##### Stop Skywire Node
```
cd $GOPATH/bin
pkill -F node.pid
```

## Open Skywire Manager View
Open [http://localhost:8000](http://localhost:8000).
The default login password for Skywire manager is **1234**.

### Conect to node
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
      -web-port :6001
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

Run this **once if you're using the official images** to change the remote repository:
```
git remote set-url origin https://github.com/skycoin/skywire.git
```
Stay up to date by updating via git:
```
cd $GOPATH/src/github.com/skycoin/skywire
git reset --hard
git clean -f -d
git pull origin master
go install -v ./...
```

Manager system image package contains Skywire Manager and a Skywire Node, other Node system image package only launch a Node.

1) Download [Manager](https://downloads.skycoin.net/skywire-images/manager.tar.gz) (IP:192.168.0.2)

2) Download [Node1](https://downloads.skycoin.net/skywire-images/node-1-03.tar.gz) (IP:192.168.0.3)

3) Download [Node2](https://downloads.skycoin.net/skywire-images/node-2-04.tar.gz) (IP:192.168.0.4)

4) Download [Node3](https://downloads.skycoin.net/skywire-images/node-3-05.tar.gz) (IP:192.168.0.5)

5) Download [Node4](https://downloads.skycoin.net/skywire-images/node-4-06.tar.gz) (IP:192.168.0.6)

6) Download [Node5](https://downloads.skycoin.net/skywire-images/node-5-07.tar.gz) (IP:192.168.0.7)

7) Download [Node6](https://downloads.skycoin.net/skywire-images/node-6-08.tar.gz) (IP:192.168.0.8)

8) Download [Node7](https://downloads.skycoin.net/skywire-images/node-7-09.tar.gz) (IP:192.168.0.9)

### Manually set IP system image

`Note: This system image only contains the basic environment of Skywire, and it needs to set IP, etc.`

Download [Pure Image](https://downloads.skycoin.net/skywire-images/skywire_pure.tar.gz)
