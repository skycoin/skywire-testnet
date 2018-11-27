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

Open a console as _root_ **on the manager** node (default IP 192.168.0.2, or your custom one). You can find some information about opening a console in the web interface in the [networking guide](https://github.com/skycoin/skywire/wiki/Networking-guide-for-the-official-router#obtain-the-ip-address-through-the-web-interface-by-opening-a-terminal-old-ui).

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

And follow the instructions in the dialog boxes, at the end you will have a file named log.txt on that folder with the log of all the operations, if you need to get that file on your pc you can [follow this procedure](https://github.com/skycoin/skywire/wiki/Backup-.skywire-folders-(public-keys)#download-backup-folders-to-your-computer-using-filezilla) to make it happen.
