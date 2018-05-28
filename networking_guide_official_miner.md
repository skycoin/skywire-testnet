![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

This is the official networking guide for configuring the router that comes with the V1 of the official skyminer router. 

## Table of Contents
* [Introduction](#introduction)
* [Requirements](#requirements)
* [Setup using the official images](#setup)
* [Usage](#usage)

## Introduction

After finishing of all the steps in this guide you will be able to access the manager's webinterface from within your home network, access the devices in the subnet of the skyminer router via SSH and of course, capable of using the SOCKS5 Skywire proxy.

The skyminer router has 8 LAN ports, so during this guide you will need to unplug one 1 from the router to gain access with your computer, later on you will have to plug it back in. 
Before you're starting with this guide please turn off all pi's, none of them need to run until you are being told to turn them on.

## Requirements in Hardware & Software

* Official skyminer router. You are advised to flash the sd cards of the orange pi prime's with the official images, see the 'Skywire Installation Guide' for instructions (https://downloads.skycoin.net/skywire/Skywire-Installation-Guide-v1.0.pdf). Manual installation of Skywire works just as fine.
* Computer/laptop with LAN port
* LAN cable

## Setup
Before you do the following steps make sure that there is no cable attached to the WAN port of the skyminer router. Restart both the skyminer router and your computer. 

### Accessing the interface of the router
Connect your computer to a LAN port of the skyminer router, it doesn't matter which one. Then open a browser window and type:
```
192.168.0.1
```
The router interface should come up, looking like this: 
![welcome_page](https://raw.githubusercontent.com/Asgaror/skywire/master/router_welcome_page.png)

If you are queried to type in a password it is 'admin'.
As you can see the default language is in chinese, to proceed we need to change it to english. To accomplish this go the rightmost tab:
![default_login_language](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/login_default_language.png)

Choose the 2nd option in the drop down menu and click on the right button. After that refresh the page, it should look like this now:

![change_language_dropdown](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/change_language.png)

### Port Forwarding rules
Now that the language is changed we can proceed and setup the necessary port forwarding rules to access the skywire manager from outside the subnet of the skyminer router (i.e. when you're not connected to one of its LAN ports).

Change to the port forwarding menu:

![choose_port_forwarding](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/choose_portforwarding.png)

Turn Port Forwarding on:

![port_forwarding_offon](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/port_forwarding_offon.png)

The screen you are now looking at looks like this:

![port_forwarding_empty](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/port_forwarding_empty.png)

Now you need to add three port forwarding rules.
Rule 1 will allow you to access the manager pi via ssh connection:
```
IP Address: 192.168.0.2
Protocol: TCP+UDP
Internal Port: 22
External Port: 22
Description: SSH
```
Click on 'Add', then proceed with rule 2.

Rule 2 will allow you to access the manager's webinterface:
```
IP Address: 192.168.0.2
Protocol: TCP+UDP
Internal Port: 8000
External Port: 8000
Description: Manager
```
Click on 'Add' and proceed with rule 3.

Rule 3 will allow you to use the SOCKS5 proxy after establishing a connection within the Skywire network:
```
IP Address: 192.168.0.2
Protocol: TCP+UDP
Internal Port: 9443
External Port: 9443
Description: SOCKS5
```
Once all rules are added it should look like this:

![port_forwarding_rules](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/port_forwarding_rules.png)

This part is finished, disconnect your computer from the LAN port and plug the remaining pi back in. 
Now please turn off the skyminer router.

### Optional: assign static ip address for the skyminer router
Now that the configuration of the skyminer router is finished you can specify a static ip for it inside your home router. This will add convenience once you want to view the manager and access the nodes during the testnet. 
To do this, you need to log into your home router (if you're not sure how to do this read this https://www.lifewire.com/how-to-find-your-default-gateway-ip-address-2626072). 

Once you're in you can go to static leases or a similar term (located somewhere in LAN settings; highly dependent on your router model, this is a very broad description. Please download the manual of your router to get a detailed guide how to accomplish this) 
and assign the static ip lease for the skyminer router. 
You will need the MAC address of the skyminer router for this, you can find it on the 'Home'page of the router interface or in 
your home router's webinterface: 

![mac_address](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/macaddress.png)

## Usage

### Obtain IP Address within your home network

Now you need to plugin a cable into the WAN port of the skyminer router going into a LAN port of your home router.

To access the manager or any device within the subnet of the skyminer router you need the ip that it got assigned by your home router. To obtain it you have 3 options (they increase in difficulty from one to three):
1) Visit the skyminer router interface and obtain the ip address from the 'Home'page. 
```
To do the following two steps you need to disconnect your computer from the skyminer router and connect it to your home router.
```
2) Login to your home router and go to 'Connected Devices', you'll see a device called 'myap', this is the skyminer router.
3) Use network scan softwares like for example nmap to scan the subnet of your home router. The following is an example for the 192.168.1.1/24 subnet
```
nmap -sP 192.168.1.1/24
```
Will give you all active devices within the subnet. 
If you aren't familiar with using the commandline you can use the multi-platform tool zenmap, which provides a UI for nmap (https://nmap.org/zenmap/).

Please note down the ip address, we'll need it to access the manager's webinterface.

### Viewing the manager 
Open a new browser window and type in the ip of the skyminer router (not the local ip, the one assigned by your home router) and specify port 8000. The following is the url if your skyminer would be at 192.168.220.146 (is most likely different from yours)
```
192.168.220.146:8000
```
Looks like this: 
![manager_login](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/manager_login.png)

Now you have to login. Default password is '1234', which you have to change immediately after
Once you're done you should be looking at the node list:

![node_list](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/node_list.jpg)

By clicking on one of the displayed nodes you'll see a page looking like this, displaying the public key and the app key of the node:

![pubkey_appkey](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/pubkey_appkey.jpg)

Make sure that everything is correct by searching for your public key on discovery.skycoin.net:8001 and under 'Settings' it should look like this:

![green_hook](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/discovery_green_hook.jpg)

If you established a connection to another node via the 'Search Services' in 'Connect to Node' it will look like this:

![connection](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/connections.png)

### Using the SOCKS5 proxy
For this to work you need to establish a connection to another node as shown in a previous screenshot. Once that is done download a proxy plugin 
to tunnel your browser traffic through the connection. The following screenshots show the settings in FoxyProxy since this is the service referred to 
in the main readme. The browser being used in the screenshots is firefox.
First choose 'options':

![options](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/options.png)

Then add a new proxy:

![add](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/add.png)

Now you can configure the proxy like this, use the ip that your home router assigned the skyminer router:

![socks5_config](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/socks5.png)

That's it, enable the rule and you can proxy your traffic through the Skywire network:

![enabled](https://raw.githubusercontent.com/Asgaror/skywire/master/networking_guide_pictures/enabled.png)




