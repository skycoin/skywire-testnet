# Upgrade scripts for Official Skyminers hardware

This are the instructions to upgrade the official Skyminers with the official hardware (Orange Pi Prime boards) There are some small bugs on the integration of the Skywire with the underlaying OS that leads to some problems with the upgrade process, this scripts will fix that and.

## Improvements & details

On this upgrade we will clear a bunch of identified bugs, see [Issue #171](https://github.com/skycoin/skywire/issues/171) for a list of them; also you get this:

* New version of the webUI
* Improved start/stop scripts, handled now by systemd the mainstream solution
* Date & time sync with mainstream internet time servers
* Updated main upgrade script (this upgrade is a one time upgrade)
* Many fixed bugs in the Skywire vs. OS integration.
* **WARNING:** the root password will be changed to ```skywire``` once you update your systems

## Provisions

* Your hardware must be configured with the default network config or you must know your manager and nodes IPs if you changed them; see the [networking guide](https://github.com/skycoin/skywire/wiki/Networking-guide-for-the-official-router) if you need to know more.
* You must have access to your manager node via a linux shell (using plain ssh on Linux/Mac or Putty on Windows)
* Your skyminer must be connected to the internet (all nodes need to be able to communicate with the outside world) or the script will fail, as we will update a few thing online.
* You must know the root password of each node (if you don't changed it, then it's ```samos```)

## Getting the upgrade

Open a console as _root_ **on the manager** node (default IP 192.168.0.2, or your custom one). Do no open a terminal in the web interface, use either Putty if you're on Windows or a terminal if you're on Mac/Linux.

Once you opened the terminal, please enter the following commands to start the upgrade:

```
cd /tmp
git clone https://github.com/skycoin/skywire.git
cd skywire/static/script
cd upgrade
```

At this point you are ready to execute the upgrade script.

## Upgrading

In the same console, run now the update script, like this:

```
./one_time_upgrade
```

Please make sure to change the root password afterwards via `passwd` command of all boards!

And follow the instructions in the dialog boxes, at the end you will have a file named log.txt on that folder with the log of all the operations, if you need to get that file on your pc you can [follow this procedure](https://github.com/skycoin/skywire/wiki/Backup-.skywire-folders-(public-keys)#download-backup-folders-to-your-computer-using-filezilla) to make it happen.

***

## Troubleshooting
All errors of the upgrade procedure are logged in a `log.txt` file, multiple upgrade attempts only append to this file. 

First upgrade attempts of the community revealed some issues:

#### ERROR: Read from socket failed: Connection reset by peer
Try to reboot the board, if this doesn't help you need to generate new rsa key pairs. 
In case you cannot access the board via GUI or the web interface of the browser, your only option left is to reflash the image of the board.

To generate new rsa keys and restart the ssh service, please execute:
```
rm -rf /etc/ssh/ssh*key.pub

ssh-keygen -A

/etc/init.d/ssh restart
```

Now your board should be accessible via SSH again.

#### ERROR: clone operation failed.
In case you encounter this error it is preceded by */tmp/upsky.sh: line 23: git: command not found*.
The solution is to install git on your system, to do this please login via SSH or open a terminal of the node in the web interface.

Proceed with installing git & gcc:
```
sudo apt-get install git gcc
```
Then manually execute the update on the node via 
```
cd /usr/local/skywire/go/src/github.com/skycoin/skywire

git reset --hard

git remote set-url origin https://github.com/skycoin/skywire.git

git pull

go install -v ./...

systemctl stop skywire-node.service

systemctl start skywire-node.service
```
