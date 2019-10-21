# Testnet Discovery Address Change Instructions
*This article outlines the necessary steps to update the discovery address of the Skywire testnet's discovery server.*

### Table Of Contents
- [Introduction](#introduction)
- [Backup your Data](#backup-your-data)
- [Official Skyminers](#official-skyminers)
- [DIY Skyminers](#diy-skyminers)
  
## Introduction

As of September 2019 the discovery address of the Skywire Testnet Discovery Server changed. Thus, all testnet participants must update the discovery address on their Skyminer (both official and DIY are affected).

The **old and invalid discovery address** is as follows:
```
discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68
```

The **new and valid discovery address** is this:

```
testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243
```

Please understand that adjusting the discovery address is mandatory if you want to continue generating the necessary uptime for being rewarded. Note that your rewards of September are **not** affected by this change!

## Backup Your Data

If you do not feel confident in performing these steps, please backup your data. 
Two backups may be performed:
- [use this guide to backup your public keys](https://github.com/SkycoinProject/skywire/wiki/Backup-Public-Keys) before you proceed.
Restoring public keys is always a quick option in case something goes wrong. 
- perform an image backup following [this guide](https://www.raspberrypi.org/magpi/back-up-raspberry-pi/) (follow *option #1*). This guide applies to any SBC and not just Raspberry Pi.

Please read the following steps of this guide very carefully and make sure you understand each steps before execution.

## Official Skyminers

These steps must be performed on each node of the Skyminer (8 nodes in total). The following are the 'requirements':
- You need a ssh client like [Putty](https://www.putty.org/) if you're on Windows to connect to the nodes (Linux/OSX users may just use the regular terminal/console)
- You must have performed the one-time-upgrade at one point (see [this article](https://github.com/SkycoinProject/skywire/#upgrade-the-presetted-system-images) for details)
- You need to know the `root` credentials of each node 
  - the default `root` password depends on the prepared image you are using:
    - official images *without* the one-time-upgrade -> `samos` (**You must perform the upgrade!**)
    - official images *with* the one-time-upgrade -> `skywire`
- Each Skyminer node must have a working internet connection

### Instructions

0. Access a node (try manager node first) as root via the network: ```ssh root@your-node-ip``` and provide the root password
0. Move to a temp folder: ```cd /tmp```
0. Get the update script: ```wget https://github.com/SkycoinProject/skywire/raw/master/static/script/upgrade/upgrade-discovery```
0. Run the script: ```bash upgrade-discovery```
0. Reboot: ```reboot```

If everything goes well you will be kicked from the terminal SSH session. Please connect to the next node and repeat these steps. Once you've performed the steps on all nodes, login to your manager and check if all of your nodes display a green status LED thus being connected and you are done. Manual confirmation of your nodes being online can be achieved by following the steps of the [Online Status Verification User Guide](https://github.com/SkycoinProject/skywire/wiki/Online-Status-Verification-User-Guide) and replacing http://discovery.skycoin.net:8001/ with http://testnet.skywire.skycoin.com:8001/#.

## DIY Skyminers

The following sections contain update instructions for our most popular DIY user guides. The title of each section is equal to the related thread title on skywug.net or of the related medium blog post.

### Skywire Systemd Service - Skywire systemd service setup guide

Taken from: [https://github.com/SkycoinProject/skywire/wiki/Skywire-Systemd-Service](https://github.com/SkycoinProject/skywire/wiki/Skywire-Systemd-Service)

Please recall that you must perform these steps on every single one of your Skyminer nodes!

#### Requirements

- You need a SSH client like [Putty](https://www.putty.org/) on Windows to connect to the nodes (Linux/OSX users may just use the regular terminal/console)
- You need to know the user credentials of each node
- Each Skyminer node must have a working internet connection

#### Instructions

0. Access a node (try the manager node first) as user `pi` or `root` via the terminal (for Linux or Mac): ```ssh pi@your-node-ip``` or using putty if you are on Windows and enter the password
0. Modify Skywire's environment path file on the nodes and manager, edit the environment path file via: ```sudo nano /etc/default/skywire```
0. Locate the old discovery address, erase it and paste the new discovery address (refer to the top of this post to identify each one, and see below for an example)
0. Once you made the change press 'ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Modify Skywire's systemd units on the nodes and manager, edit the service files via: ```sudo nano $GOPATH/src/github.com/SkycoinProject/skywire/static/script/node_start```
0. Locate the old discovery address, erase it and paste the new discovery address environment path variable `${DISCOVERY_ADDR}`
0. Once you made the change press 'ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If everything goes well you will be kicked from the terminal SSH session. Please continue with the next node and repeat these steps. Once you've performed the steps on all nodes, login to your manager and check if all of your nodes display a green status LED thus being connected and you are done. Manual confirmation of your nodes being online can be achieved by following the steps of the [Online Status Verification User Guide](https://github.com/SkycoinProject/skywire/wiki/Online-Status-Verification-User-Guide) and replacing http://discovery.skycoin.net:8001/ with http://testnet.skywire.skycoin.com:8001/#.

#### Example

Your original ```/etc/default/skywire``` file's relevant lines look like the following:

```
SKYWIRE_START_CMD=${SKYWIRE_DIR}/static/script/start
Web_Dir=${SKYWIRE_DIR}/static/skywire-manager
SKYWIRE_GIT_URL="https://github.com/SkycoinProject/skywire.git"
DISCOVERY_ADDR=discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68
```

Modified lines must look like this:

```
SKYWIRE_START_CMD=${SKYWIRE_DIR}/static/script/start
Web_Dir=${SKYWIRE_DIR}/static/skywire-manager
SKYWIRE_GIT_URL="https://github.com/SkycoinProject/skywire.git"
DISCOVERY_ADDR=testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243
```

Your original ```node_start``` file's relevant lines look like the following:

```
# start routine only node
cd ${GOPATH}/bin/
nohup ./skywire-node -connect-manager -manager-address ${MANAGER_IP}:5998 -manager-web ${MANAGER_IP}:8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001  > /dev/null 2>&1  &
echo $! > "${Node_Pid_FILE}"
```

Modified lines must look like this:

```
# start routine only node
cd ${GOPATH}/bin/
nohup ./skywire-node -connect-manager -manager-address ${MANAGER_IP}:5998 -manager-web ${MANAGER_IP}:8000 -discovery-address ${DISCOVERY_ADDR} -address :5000 -web-port :6001  > /dev/null 2>&1  &
echo $! > "${Node_Pid_FILE}"
```


### Raspberry Pi Skyminer Setup For Noobs

Taken from our user forum skywug.net: [https://skywug.net/forum/Thread-Raspberry-Pi-Sky-Miner-Setup-for-noobs-TESTNET-READY?highlight=DIY+miner](https://skywug.net/forum/Thread-Raspberry-Pi-Sky-Miner-Setup-for-noobs-TESTNET-READY?highlight=DIY+miner)

Please recall that you must perform these steps on every single one of your Skyminer nodes!

#### Requirements
- You need a SSH client like [Putty](https://www.putty.org/) on Windows to connect to the nodes (Linux/OSX users may just use the regular terminal/console)
- You need to know the user `pi` credentials to login to each node

#### Instructions

0. Access a node (try the manager node first) as user `pi` via the network: ```ssh pi@your-node-ip``` and enter the password
0. Change to the Skywire folder: ```cd /home/pi/go/src/github.com/SkycoinProject/skywire```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/SkycoinProject/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Locate and edit the file 'startsecond.sh', issuing this command will suffice: ```cd ~/ && nano startsecond.sh```
0. Locate the old discovery address, erase it and paste the new discovery address (refer to the top of this post to identify each one, and see below for an example)
0. Once you made the change press 'ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If everything goes well you will be kicked from the terminal SSH session. Please continue with the next node and repeat these steps. Once you've performed the steps on all nodes, login to your manager and check if all of your nodes display a green status LED thus being connected and you are done. Manual confirmation of your nodes being online can be achieved by following the steps of the [Online Status Verification User Guide](https://github.com/SkycoinProject/skywire/wiki/Online-Status-Verification-User-Guide) and replacing http://discovery.skycoin.net:8001/ with http://testnet.skywire.skycoin.com:8001/#.

#### Example

Your original lines in ```startsecond.sh``` which are relevant to this change look like this ("some_ip_here" refers to your manager ip address: don't change it!)

```
cd /home/pi/go/bin
nohup ./node -connect-manager -manager-address "some_ip_here":5998 -manager-web "some_ip_here":8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001 > /dev/null 2>&1 &  
```

Modified lines must look like this:

```
cd /home/pi/go/bin
nohup ./node -connect-manager -manager-address "some_ip_here":5998 -manager-web "some_ip_here":8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 > /dev/null 2>&1 &  
```

### DIY Miner - A Complete Guide For Hardware/Software Configuration

Taken from: [https://skywug.net/forum/Thread-DIY-Miner-A-complete-guide-for-Hardware-Software-configuration](https://skywug.net/forum/Thread-DIY-Miner-A-complete-guide-for-Hardware-Software-configuration)

Please recall that you must perform these steps on every single one of your Skyminer nodes!

#### Requirements

- You need a SSH client like [Putty](https://www.putty.org/) on Windows to connect to the nodes (Linux/OSX users may just use the regular terminal/console)
- You need to know the user `root` credentials of each node
- Each Skyminer node must have a working internet connection

#### Instructions

0. Access a node (try the manager node first) as user `pi` via the network: ```ssh pi@your-node-ip``` and enter the password
0. Change to `root`: ```sudo bash```
0. Change to the Skywire folder: ```cd $GOPATH/src/github.com/SkycoinProject/skywire/```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/SkycoinProject/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Modify Skywire's systemd units on the nodes (not manager), edit the service file via: ```nano /etc/systemd/system/skynode.service```
0. Locate the old discovery address, erase it and paste the new discovery address (refer to the top of this post to identify each one, and see below for an example)
0. Once you made the change press 'ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If everything goes well you will be kicked from the terminal SSH session. Please continue with the next node and repeat these steps. Once you've performed the steps on all nodes, login to your manager and check if all of your nodes display a green status LED thus being connected and you are done. Manual confirmation of your nodes being online can be achieved by following the steps of the [Online Status Verification User Guide](https://github.com/SkycoinProject/skywire/wiki/Online-Status-Verification-User-Guide) and replacing http://discovery.skycoin.net:8001/ with http://testnet.skywire.skycoin.com:8001/#.

#### Example

Your original ```skywire.service``` file's relevant lines look like the following ("some_ip_here" refers to your manager ip address: don't change it!)

```
Environment="GOPATH=/root/go" "GOBIN=$GOPATH/bin"
ExecStart=/root/go/bin/skywire-node -connect-manager -manager-address ip_of_manager:5998 -manager-web ip_of_manager:8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001
ExecStop=kill
```

Modified lines must look like this:

```
Environment="GOPATH=/root/go" "GOBIN=$GOPATH/bin"
ExecStart=/root/go/bin/skywire-node -connect-manager -manager-address ip_of_manager:5998 -manager-web ip_of_manager:8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001
ExecStop=kill
```

### Ronny’s Cheap Man’s Skyminer a Skyminer for only 40 bucks in less than 1 hour

Taken from here: [https://medium.com/coinmonks/build-the-ronnys-cheap-man-s-skyminer-a-skyminer-for-only-40-bucks-in-less-than-1-hour-526714fe7a3a](https://medium.com/coinmonks/build-the-ronnys-cheap-man-s-skyminer-a-skyminer-for-only-40-bucks-in-less-than-1-hour-526714fe7a3a)

If you didn't update your setup after following Ronny's guide it's time to do it as the following is based on the [**updated** version of this DIY miner](https://medium.com/@CryptoRonny/updating-the-ronnys-cheap-man-s-miner-f4999278c262). Please follow the link for more information.

The steps of this guide are just for one singular node that runs both, the node & the manager. A Raspberry Pi servers as SBC platform.

#### Requirements

- You need a SSH client like [Putty](https://www.putty.org/) on Windows to connect to the nodes (Linux/OSX users may just use the regular terminal/console)
- You need to know the user `pi` credentials to get into the node
- Each Skyminer node must have a working internet connection

#### Instructions

0. Access the node as user `pi` via the network: ```ssh pi@your-node-ip``` and enter the password
0. Change to `root`: ```sudo bash```
0. Change to the Skywire folder: ```cd $GOPATH/src/github.com/SkycoinProject/skywire/```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/SkycoinProject/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Modify Skywire's start script open it for edit: ```nano /etc/init.d/MyScript.sh```
0. Locate the old discovery address, erase it and paste the new discovery address (refer to the top of this post to identify each one, and see below for an example)
0. Once you made the change press 'ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If everything goes well you will be kicked from the terminal SSH session. Please continue with the next node and repeat these steps. Once you've performed the steps on all nodes, login to your manager and check if all of your nodes display a green status LED thus being connected and you are done. Manual confirmation of your nodes being online can be achieved by following the steps of the [Online Status Verification User Guide](https://github.com/SkycoinProject/skywire/wiki/Online-Status-Verification-User-Guide) and replacing http://discovery.skycoin.net:8001/ with http://testnet.skywire.skycoin.com:8001/#.

#### Example

Your original ```MyScript.sh``` file's relevant lines look as follows:  

```
cd $GOPATH/bin
./skywire-node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001 &> /dev/null 2>&1 &
echo "Skywire monitor started."
```

The modified lines must look like this:

```
cd $GOPATH/bin
./skywire-node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 &> /dev/null 2>&1 &
echo "Skywire monitor started."
```
