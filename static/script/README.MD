# SkyWire Linux Image Scripts

## Network policies that impact on the scripts

The skyminer network will always use this policies:

* We will use a 192.168.0.0/24 network (netmask 255.255.255.0) by default
* Some of the last digit of the IPs in the network are specials:
  * 192.168.0.**1** is the router/gateway
  * 192.168.0.**2** is the manager and node number 0
  * 192.168.0.**x** are the rest of the nodes (from .3 up to .254, keep them in consecutive order)
* The unix start scripts will look for this last digits to do its magic.

If you run a custom skywire setup with different IP range it's adviced to keep the gateway as .1 and the manager in .2 for this to work, or you will need to change a lot of things in the scripts/configuration to make it work properly, you have been warned.

## Start script

The main start script is ```skywire/static/start``` and it will look for the ending digit of the IP address and will start the manager and/or node in each case. This is acomplished by calling one of the two scripts named ```manager_start``` or ```node_start```

## ENV vars

The skywire apps needs some vars on the environment to work properly, so far this is the most important ones.

```sh
GOPATH=/usr/local/skywire/go
GOROOT=/usr/local/go
```

This and other vars are handeld by using the "." (include from file) feature of unix systems, the file with the vars in in ```skywire/static/skywire.defaults``` this script are copied to ```/etc/default/skywire``` on the fisrt run.

From that point forward all unix scripts will use it (include) at top of the scripts to get sure we have all the env vars needed for all.
