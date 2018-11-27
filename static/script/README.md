# SkyWire Linux Image Scripts

In this file you can find some of the elements for a successful integration of the skywire apps to the underlaying linux OS (at his moment Orange Pi Prime on the official SkyWire images).

## ENV vars

The skywire apps needs some environment vars to work properly, this is acomplished by the "." _(include from file)_ feature of linux systems, the file that hold the vars can be found on ```static/script/skywire.defaults``` this file is copied to ```/etc/default/skywire``` on the first run of the update, manager or node start.

So, all linux scripts must use it (include) at top of the scripts to get sure we have all the env vars needed for a properly Skywire operation.

If you need to modify any of the vars in the ```static/script/skywire.defaults```, for example if you use a different IP set you will need to set the MANAGER_IP var to your managers's IP. In this case you will need to erase the ```/etc/default/skywire``` file to update it, on the next skywire startup it will be updated. 

## Network policies that impact on the scripts

The skyminer network will always use this policies:

* We will use a 192.168.0.0/24 network (netmask 255.255.255.0) by default
* Some of the last digit of the IPs in the network are specials:
  * 192.168.0.**1** is the router/gateway.
  * 192.168.0.**2** is the manager and first node.
  * 192.168.0.**x** are the rest of the nodes (from .3 up to .254, keep them in consecutive order)
* The linux start scripts will look for this last digits to do its magic.

If you run a custom skywire setup with different IP range it's adviced to keep the gateway as .1 and the manager in .2 for this to work, or you will need to change a few things in the scripts/configuration to make it work properly (like the MANAGER_IP in ```static/script/skywire.defaults```, etc.) you have been warned.

## Start & stop scripts

The main start script is in ```static/script/start``` it will look for the ending digit of the IP address and will call one of the two scripts named ```static/script/manager_start``` or ```static/script/node_start``` in each case.

Of curse, you can call the ```static/script/manager_start``` or ```static/script/node_start``` directly if you need it, like on the case of a different IP set when you can't use the default network policy (.2 to manager, etc.)

There is also an all case stop script in ```static/script/stop``` if you need it.

## Systemd integration

The scripts ```static/script/manager_start```, ```static/script/node_start``` and ```static/script/stop``` are systemd friendly, in fact you have the two units files for systemd in the folder ```static/script/upgrade/data/```, look for the .service files and see main readme for instructions on how to install & use them.

## Main update script

The main update script is located in ```static/script/update``` it will check differences between the local and remote git repo heads; if there is any difference it will stop the services, update the local git respo, compile and install the apps and then restart the services.

If you hit any troubles check /tmp/skywire-info/skywire_install_errors.log for details about errors/details.

## One time upgrade script for official skyminers

There is also a one time upgrade script in the folder ```static/script/upgrade/``` that you need to apply to upgrade de official skyminer OS images for the OrangePi Prime, this must be used just one time & before the main upgrade script.
