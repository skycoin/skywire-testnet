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

First take a look at the [script integration README](static/script/README.md) to know a few facts ant tips that will help you to understand how Skywire is integrated to the Unix systems, very important is the part of the Network Policies.

Now if you read that you must realize that if you use a different IP set you will need to change a few things, we will point them out when needed.

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

#### Set the IP of the manager

If you are using the default network IP set you are set, follow to the next step.

If you uses a different IP set you need to modify the file in ```static/script/skywire.defaults```, in particular the variable called ```MANAGER_IP``` in the default file it points to the default manager IP, in the case of a different IP set this will need to be changed to the manager IP.

Just for a matter of precaution, after modify this file be sure that there isn't a fille called ```/etc/default/skywire``` if it's there erase it. It will be updated once you run skywire.

## Run Skywire

### Unix systems

#### Run Skywire Manager
```
cd $GOPATH/bin
./skywire-manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

`tip: If you run with the above command, you will not be able to close the current window or you will close Skywire Manger.`

If you need to close the current window and continue to run Skywire Manager, you can use
```
cd $GOPATH/bin
nohup ./skywire-manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager > /dev/null 2>&1 &sleep 3
```

`Note: do not execute the above two commands at the same time, just select one of them.`

#### Run Skywire Node

Open a new command window

```
cd $GOPATH/bin
./skywire-node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 
```

`tip: If you run with the above command, you will not be able to close the current window or you will close Skywire Node.`

If you need to close the current window and continue to run Skywire Manager, you can use
```
cd $GOPATH/bin
nohup ./skywire-node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 > /dev/null 2>&1 &cd /
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


### Official Images

#### Run Skywire Manager

Open a command window on a PC that will act like a manager and follow the install procedure, then to start a node do this:

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/manager_start
```

`tip: the manager start script will also run a local node, you don't need to run in manually on the manager.`

#### Run Skywire Node

Open a command window on a node only computer and follow the install procedure, then to start a node:

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

###### Installing & start of mananger unit file on systemd 

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-manager.service /etc/systemd/system/
systemctl enable skywire-manager
systemctl start skywire-manager
```

###### Installing & start of nodes unit file on systemd 

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-node.service /etc/systemd/system/
systemctl enable skywire-node
systemctl start skywire-node
```

From this point forward you can user this services to start/stop your skywire instances via systemd commands:

```
# for the nodes
systemctl *start* skywire-node
systemctl *stop* skywire-node
systemctl *status* skywire-node
# for the manager
systemctl *start* skywire-manager
systemctl *stop* skywire-manager
systemctl *status* skywire-manager
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
      -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243
```

### Docker Compose

```
docker-compose up
```

Open [http://localhost:8000](http://localhost:8000).

## Download System Images

<a name="images"></a>

Note: these images can only be run on [Orange Pi Prime](http://www.orangepi.cn/OrangePiPrime/index_cn.html).

### Skyflash & Skybian
We developed our own custom flashing tool that prepares & flashes our custom OS [Skybian](https://github.com/skycoin/skybian) for operation on our Skyminers. Skybian is our custom OS built upon armbian. It comes with Skywire and its dependencies preinstalled and its IP configuration is adjusted by [Skyflash](https://github.com/skycoin/skyflash) according to your network environment. Please refer to the [installation guide](https://github.com/skycoin/skywire/wiki/Skyminer-Skywire-installation-guide#installation) on our wiki for more details & instructions.


