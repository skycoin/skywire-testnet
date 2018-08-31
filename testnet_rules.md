![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

### Skywire Testnet Rules

*Disclaimer: All information about the rules in this post can be found on various posts on our [blog](skycoin.net/blog) or on our [medium](medium.com/skycoin). The rewards in this post are subject to change. Updates in this post will be followed by a notification via the [official Skywire PSA channel](https://t.me/SkywirePSA) on telegram.*

#### Table of Contents
* [Introduction](#introduction)
* [Rules](#rules)
* [Rewards](#rewards)
* [Hardware ](#hardware)
* [Whitelist](#whitelist)

## Introduction
This article represents the central source of information for the ongoing Skywire testnet. All information about rewards and potential changes will be published here, so check in regularely.

Read this information thoroughly and ask in the [Skywire](https://t.me/skywire) telegram channel if some things are not covered. 

***

## Rules

**Each whitelisted person is eligible of receiving rewards for 1 official miner or 1 DIY miner with up to 8 nodes, in the case that either the official or the DIY miner are running at a different location the person will receive rewards for both if the requirement is met.**

* The demand for a different location is due to the fact that we want to spread out the meshnet and not concentrate the location of nodes to specific central points, which would result in paying people to run orange pi's and not to reward them for providing a useful service to the network.

* *Submitting applications under multiple email addresses is illegal and likely to be detected - measures will be taken if such actions are recognized.*


## Rewards

### Facts

**Eligible for rewards are only the whitelisted nodes, that meet the 75% uptime requirement during the month. It doesn't matter in which week you're getting whitelisted, you're uptime is being counted regardless of that.**

As previously stated 1 person can receive rewards for 1 official miner and 1 DIY miner if one of them is in a different location than the other.

The rewards are paid every month around the 5th or with a delay, depending if other things delay the process.
They are paid on a node by node basis and are subject to change, meaning they may be different for the months to come. 

The total amount of rewards per month are **25,000 Skycoin**, divided into two separate pools
* **15,000 Skycoin for the official miners**
* **10,000 Skycoin for the whitelisted DIY miners**

For now, the rewards for each node that meets [the requirement](https://github.com/skycoin/skywire/wiki/_new#requirements) are as follows:

* **DIY: 6 Skycoin / node with a maximum of 8 nodes; 48 Skycoin maximum**

* **Official miner: 12 Skycoin / node**

*Faulty orange pi's from official miners will be rewarded regardless of your uptime until you receive a replacement. If the replacement doesn't arrive in time for you to make the uptime requirement because it arrived on short notice or not on the schedule at all you will be rewarded as well. Since we are taking care of this manually you are requested to contact one of our team members (@asxtree @MrHodlr @Paperstream).*

As soon as the pool size would be surpassed we will adjust the rewards: We will split up the Skycoin in the pool evenly between all nodes that are eligible for rewards, i.e. the total amount is shared amongst all nodes which meet the uptime requirements, with a maximum of 6 Skycoin per node (DIY) and 12 Skycoin per node (officia).

### Requirement

**You need to have 75% uptime during the month you want to be rewarded for.** It doesn't matter in which week you're getting whitelisted, you're getting accounted for your uptime regardless.

As of now, you are provided with two tools to check whether or not you're online and generating uptime:
* The [discovery website](http://discovery.skycoin.net:8001/)
* The [node checker tool](http://167.99.207.153:8001/)

Additional to that you should look in the Skywire manager web interface for two things:

#### For a running node app, which is marked by the red rectangle where it says 'app key'

![node_app](https://raw.githubusercontent.com/Asgaror/skywire/binary_data_storage/pictures/testnet%20guideline/node_app.png)

#### For a green check mark next to the discovery address on each node under 'Settings':

![green_check_mark](https://raw.githubusercontent.com/Asgaror/skywire/binary_data_storage/pictures/testnet%20guideline/discovery_address.png)

* Make sure that you are connected to **discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68** as this is the discovery server that is used by us to calculate the uptime.

***

## Hardware

**VM's, servers or personal computers are not allowed in the testnet, i.e. they will not be whitelisted and receive rewards.** 

The following hardware is allowed and can be chosen on the [whitelist application form](https://www.skycoin.net/whitelist/):

#### Orange Pi
     - Prime
     - 2G-IOT
     - 4G-IOT
     - i96
     - Lite
     - Lite2
     - One
     - One-Plus
     - PC
     - PC-Plus
     - PC2
     - Plus
     - Plus2
     - Plus2E
     - RK3399
     - Win
     - Win-Plus
     - Zero
     - Zero-Plus
     - Zero-Plus2

#### Raspberry Pi
     - 1-Model-A+
     - 1-Model-B+
     - 2-Model-B
     - 3-Model-B
     - 3-Model-B+
     - Zero-W
     - Zero

#### Asus
     - Tinkerboard

#### Banana Pi
     - BPI-D1
     - BPI-G1
     - BPI-M1
     - BPI-M1+
     - BPI-M2
     - BPI-M2+
     - BPI-M2-Berry
     - BPI-M2M
     - BPI-M2U
     - BPI-M64
     - BPI-R2
     - BPI-R3
     - BPI-Zero

#### Beelink
     - X2

#### Cubieboard
     - Cubietruck
     - Cubietruck-Plus
     - 1
     - 2
     - 4

#### Helios
     - 4

#### Libre Computer
     - Le-Potato-AML-S905X-CC
     - Renegade-ROC-RK3328-CC
     - Tritium-ALL-H3-CC

#### MQMaker
     - MiQi

#### NanoPi
     - NanoPi
     - 2
     - 2-Fire
     - A64
     - K2
     - M1
     - M1-plus
     - M2
     - M2A
     - M3
     - NEO
     - NEO-Air
     - NEO-Core
     - NEO-Core2
     - NEO2
     - S2
     - Smart4418

#### Odroid
     - HC1
     - HC2
     - MC1
     - XU4

#### Olimex
     - Lime1
     - Lime2
     - Lime2-eMMC
     - LimeA33
     - Micro

#### Pine
     - Pine-A64
     - Pinebook-A64
     - Sopine-A64
     - Rock64

#### SolidRun
     - CuBox-i
     - CuBox-Pulse
     - Humming-Board
     - Humming-Board-Pulse
     - ClearCloud-8K
     - ClearFog-A38
     - ClearFog-GT-8K

#### Udoo
     - Blu
     - Bricks
     - Dual
     - Neo
     - Quad
     - X86

**If you like to use other boards please contact the team first to be approved before you buy them, only the boards on the list are guaranteed to be whitelisted.**

***

## Whitelist

The whitelist form can be found at [skycoin.net/whitelist](skycoin.net/whitelist).

### July

    - LAST WHITELISTED QUEUE POSITION: 411
    - LAST WHITELISTED APPLICATION DATE (UTC+8): 2018-05-31 21:18:02
    - AMOUNT OF WHITELISTED OFFICIAL MINERS: 360

#### Updates
    - The whitelist is updated on a monthly basis, meaning that we are whitelisting 200 applications each month.
    - The numbers above are getting updated on a monthly basis as well

#### Position
    - You can request your actual queue position by contacting one of our team members on telegram
    - Calculate your individual waiting period based on this queue position
    - The application ID is not representative of your queue position
    - Multiple submitted applications by the same email each have an application ID
    
#### Get your data
    - Contact @asxtree @MrHodlr @Paperstream to obtain your queue position, your submitted data etc.
    - You can double check with our team members (see above) if we received your application.
    - Unlike the whitelist data, the team receives *weekly updates* on the application data


### Facts

* The whitelist is not going to be updated on a weekly basis as previously stated but on a monthly basis.
* The whitelist is a queue based on a first come first serve basis. Each month we are & have been whitelisting 200 applications, the benchmark for applications is the hardware list above + the official miner specifications.
* It doesn't matter in terms of rewards in which week you're getting whitelisted.
* Your spot is recognized by your email address, you can think of that just as you would think of an account. Right now we have no account system for you in place to check your spot, provided bandwidth etc. but this will be done for the mainnet.
* Official miners need to submit the form using their purchasing email, as this is the only way for us to identify them 
* Official miners are whitelisted by default, meaning that they are whitelisted as soon as they submit the application form.
* You are advised to [backup your public keys](https://github.com/skycoin/skywire/wiki/Backup-.skywire-folders-(public-keys)) but if something happens and you have to reflash then simply resubmit the application form including all current active public keys.
* If you need to update the Skycoin wallet address for receiving the rewards please submit the form again, send support@skycoin.net an email notification about the change and contact someone on telegram if you don't receive an answer after some days.
* For email address changes of official miners please send the team an email at support@skycoin.net and contact someone on telegram if you don't receive an answer after some days.
* Make sure to generate new public keys after ownership transfer on an official miner and to resubmit the [whitelist application form](skycoin.net/whitelist)

### The form
The following information needs to be submitted for each type of miner.

#### Official Miner
     - Name
     - Purchasing email address
     - Telegram account (optional; you should join, there is an exclusive official miner chat waiting for you)
     - The city doesn't have to be 100% precise
     - Skycoin wallet address
     - Your 8 public keys. Simply submit 7 public keys if you have a faulty pi (reward will be paid regardless).

#### DIY Miner
     - Name
     - Email address
     - Telegram account (optional; you should join, there is an awesome community waiting for you)
     - The city doesn't have to be 100% precise
     - Skycoin wallet address
     - Node quantity: The number of pis you're running in your miner
     - Node Hardware: Specify the hardware you're using. Add a note if you have merged more than 1 type of board in your miner
     - Node OS: The OS you're running on the pis
     - Node brief description: Describe your setup, the router you're using & the things that you will present on the pictures
     - Miner photos: At least three photos, each from a different perspective (each one max 3MB in size)
     - Your public keys
