# SkyWire Linux Image Scripts

In this file you can find some of the elements for a successful integration of the skywire apps to the underlaying linux OS (at his moment Orange Pi Prime on the official SkyWire images).

## ENV vars

The skywire apps needs some vars set right on the environment to work properly, this is acomplished by the "." (include from file) feature of unix systems, the file that hold the vars can be found on ```skywire/static/skywire.defaults``` this file is copied to ```/etc/default/skywire``` on the fisrt run.

So, all unix scripts must use it (include) at top of the scripts to get sure we have all the env vars needed for a properly SkyWire operation.

## Network policies that impact on the scripts

The skyminer network will always use this policies:

* We will use a 192.168.0.0/24 network (netmask 255.255.255.0) by default
* Some of the last digit of the IPs in the network are specials:
  * 192.168.0.**1** is the router/gateway.
  * 192.168.0.**2** is the manager and first node.
  * 192.168.0.**x** are the rest of the nodes (from .3 up to .254, keep them in consecutive order)
* The unix start scripts will look for this last digits to do its magic.

If you run a custom skywire setup with different IP range it's adviced to keep the gateway as .1 and the manager in .2 for this to work, or you will need to change a lot of things in the scripts/configuration to make it work properly, you have been warned.

## Start & stop scripts

The main start script is ```skywire/static/start``` it will look for the ending digit of the IP address and will start the manager and/or node in each case. This is acomplished by calling one of the two scripts named ```manager_start``` or ```node_start```

There is also a stop script in ```skywire/static/start``` if you need it, also, it's used on the update process.

## Update script

The update script is located in ```skywire/static/update``` it will check differences between the local and remote git repo heads; if there is any difference it will stop the services, update the local respository, compile and install the apps and then restart the services.

If you hit any troubles check /tmp/skywire-info/skywire_install_errors.log for details about errors/details.
