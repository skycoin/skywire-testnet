# Tesnet discovery address change

As of September 2019 the discovery address changed, so you need to update the discovery address on your Skyminer (official or DIY)

The former discovery address was this:

```
discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68
```

The new discovery address is this:

```
testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243
```

## Warning: if you are unsure or in doubt make a backup of your SD card

If you are not confident in your skills to do this steps please do backup your SD cards (all of them) before proceed, in that way you can always return to the last stable state in case of error or problems

You can use the [Raspberry Pi Guide](https://www.raspberrypi.org/magpi/back-up-raspberry-pi/) to make it, but please take note that you are **only for option #1** on that guide, also note that this Guide's option #1 apply to any SBC that uses SD cards, not just Rapsberrys.

Even so, please read carefully and make sure you understand the steps before proceed

## Change in the official Skyminer

This steps must be done for each node in the skyminer (8 in total) this is a check list to do the job follows:

- You need a ssh client like [Putty](https://www.putty.org/) in Windows to connect to the nodes (in linux/OSX just use the console)
- You need to know the root credentials to get in each node
- The Skyminer node needs to be connected to the internet for this to work
- Your skyminers need to be [upgraded](https://github.com/skycoin/skywire/#upgrade-the-presetted-system-images) and not just the plain images, if you did the upgrade back then, your are ready to go if not then you need to doit now, click the 'upgraded' link to know more.

### Hands to work

0. Access a node (try manager node first) as root via the network: ```ssh root@your-node-ip``` and provide the root password
0. Move to a temp folder: ```cd /tmp```
0. Get the update script: ```wget https://github.com/skycoin/skywire/raw/master/static/script/upgrade/upgrade-discovery```
0. Run the script: ```bash upgrade-discovery```
0. Reboot: ```reboot```

If all goes well you will get disconnected, connect to the next node and repeat the exact steps; at the end login into your manager nad check all nodes is on green and connected, you are done.

## Change in DIY miners

As there are a few guides to setup a DIY miner I will try to give directions for the most common used guides out there

### Raspberry Pi Sky Miner Setup for noobs (skywug.net)

From here: [https://skywug.net/forum/Thread-Raspberry-Pi-Sky-Miner-Setup-for-noobs-TESTNET-READY?highlight=DIY+miner](https://skywug.net/forum/Thread-Raspberry-Pi-Sky-Miner-Setup-for-noobs-TESTNET-READY?highlight=DIY+miner)

Remember you need to do this on each Rasberry Pi in your DIY miner

#### Check list

- You need a ssh client like [Putty](https://www.putty.org/) in Windows to connect to the nodes (in linux/OSX just use the console)
- You need to know the user pi credentials to get in each node

#### Steps

0. Access a node (try manager node first) as user pi via the network: ```ssh pi@your-node-ip``` provide the password
0. Change to the Skywire folder: ```cd /home/pi/go/src/github.com/skycoin/skywire```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/skycoin/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Locate and edit the file 'startsecond.sh' issuing this command will suffice: ```cd ~/ && nano startsecond.sh```
0. Locate the old discovery address, erase it and paste the new discovery address (see top of this post to identify each one, and see below for an example)
0. Once you made the change press 'Ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If all goes well you will get disconnected, connect to the next node and repeat the exact steps, at the end login into your manager nad check all nodes is on green and connected, you are done

#### Example:

Your original ```startsecond.sh``` relevant lines looks like this ("some_ip_here" refers to your manager ip address: don't touch it )

```
cd /home/pi/go/bin
nohup ./node -connect-manager -manager-address "some_ip_here":5998 -manager-web "some_ip_here":8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001 > /dev/null 2>&1 &  
```

Modified lines must look like this:

```
cd /home/pi/go/bin
nohup ./node -connect-manager -manager-address "some_ip_here":5998 -manager-web "some_ip_here":8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 > /dev/null 2>&1 &  
```

### DIY Miner - A complete guide for Hardware/Software configuration (skyguw.net)

From here: [https://skywug.net/forum/Thread-DIY-Miner-A-complete-guide-for-Hardware-Software-configuration](https://skywug.net/forum/Thread-DIY-Miner-A-complete-guide-for-Hardware-Software-configuration)

Remember: you need to do this on each node!

#### Check list

- You need a ssh client like [Putty](https://www.putty.org/) in Windows to connect to the nodes (in linux/OSX just use the console)
- You need to know the user root credentials to get in each node
- The node needs to be connected to the internet for this to work

#### Steps

0. Access a node (try manager node first) as user pi via the network: ```ssh pi@your-node-ip``` and provide the password
0. Change to root: ```sudo bash```
0. Change to the Skywire folder: ```cd $GOPATH/src/github.com/skycoin/skywire/```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/skycoin/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Modify Skywire's systemd units on the nodes (not manager), open it for edit: ```nano /etc/systemd/system/skynode.service```
0. Locate the old discovery address, erase it and paste the new discovery address (see top of this post to identify each one, and see below for an example)
0. Once you made the change press 'Ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If all goes well you will get disconnected, connect to the next node and repeat the exact steps, at the end login into your manager nad check all nodes is on green and connected, you are done

#### Example:

Your original ```skywire.service``` file's relevant lines looks like this ("some_ip_here" refers to your manager ip address: don't touch it )

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

### Ronny’s Cheap Man’s Skyminer a Skyminer for only 40 bucks in less than 1 hour (medium.com)

From here: [https://medium.com/coinmonks/build-the-ronnys-cheap-man-s-skyminer-a-skyminer-for-only-40-bucks-in-less-than-1-hour-526714fe7a3a](https://medium.com/coinmonks/build-the-ronnys-cheap-man-s-skyminer-a-skyminer-for-only-40-bucks-in-less-than-1-hour-526714fe7a3a)

If you don't updated your Ronny's setup it's time to do it as the following is based on the [**updated** version of this DIY miner](https://medium.com/@CryptoRonny/updating-the-ronnys-cheap-man-s-miner-f4999278c262) click the link to know how to doit

This guide is just for one node that has inside the node & the manager using a Raspberry Pi as a SBC

#### Check list

- You need a ssh client like [Putty](https://www.putty.org/) in Windows to connect to the nodes (in linux/OSX just use the console)
- You need to know the user pi credentials to get into the node
- The node needs to be connected to the internet for this to work

#### Steps

0. Access the node as user pi via the network: ```ssh pi@your-node-ip``` and provide the password
0. Change to root: ```sudo bash```
0. Change to the Skywire folder: ```cd $GOPATH/src/github.com/skycoin/skywire/```
0. Clean any local modified files and reset to default version: ```git checkout master && git reset --hard && git clean -f -d```
0. Update Skywire to the latest version: ```git pull https://github.com/skycoin/skywire.git```
0. Compile and install the latest changes in the sources: ```cd cmd && go install ./...```
0. Modify Skywire's start script open it for edit: ```nano /etc/init.d/MyScript.sh```
0. Locate the old discovery address, erase it and paste the new discovery address (see top of this post to identify each one, and see below for an example)
0. Once you made the change press 'Ctrl+x' and then 'y' and hit enter to confirm and save the changes you made
0. Reboot the node: ```sudo reboot```

If all goes well you will get disconnected, connect to the next node and repeat the exact steps, at the end login into your manager nad check all nodes is on green and connected, you are done

#### Example:

Your original ```MyScript.sh``` file's relevant lines looks like this:  

```
cd $GOPATH/bin
./skywire-node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68 -address :5000 -web-port :6001 &> /dev/null 2>&1 &
echo "Skywire monitor started."
```

Modified lines must look like this:

```
cd $GOPATH/bin
./skywire-node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address testnet.skywire.skycoin.com:5999-028ec969bdeb92a1991bb19c948645ac8150468a6919113061899051409de3f243 -address :5000 -web-port :6001 &> /dev/null 2>&1 &
echo "Skywire monitor started."
```
