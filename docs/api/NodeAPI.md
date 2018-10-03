# Skywire Node API Documentation
**Note: This document is a work in progress**

The following describes the Web API for the Skywire Node (`node`) application. You will need access to a running instance of both the `manager` and `node`  applications in order to utilise the APIs. Node: Some `node` APIs first require authorisation from the `manager`.

Examples provided below assume the Node is running on the local machine (127.0.0.1). The default port for accessing the API is `6001`. 
All Node and Application keys have been deliberatly altered to ensure they are invalid.

## Node API
The following API services are made avaiable by the Skywire Node application (`node`):
- [NODE](#node)
    - [Get Node Signature](#get-node-signature)
    - [Get Node Information](#get-node-information)
    - [Get Node Message](#get-node-message)
    - [Get Node Applications](#get-node-applications)
    - [Reboot Node](#reboot-node)
- [RUN](#run)
    - [Run SSHS](#run-sshs)
    - [Run SSHC](#run-sshc)
    - [sockss](#run-sockss)
    - [socksc](#run-socksc)
    - [Update](#run-update)
    - [Check Update](#run-checkUpdate)
    - [Run Shell](#run-shell)
    - [Run Command](#run-cmd)
    - [Get Shell Outpot](#get-shell-output)
    - [Search for Services](#search-for-services)
    - [Search for Services Results](#search-for-services-results)
    - [Set Autostart Config](#set-autostart-config)
    - [Close Application](#close-application)
    - [TERM](#run-term)


### Get Node Signature
#### Usage
```
URI: /node/getSig
Method: TBA
```
Example:
```sh
```

Response:
```json
```

### Get Node Information
Get information about the specific node the request is being made against.


You must successfully `/login` on the Manager to aquire a session cookie before calling this API. A valid Manager session cookie must be passed along with this request.

#### Usage
```
URI: /node/getInfo
Method: Get
```

Request:
```sh
curl "http://127.0.0.1:6001/node/getInfo?token=ca51143c60b1ab2078cacd619f1c4f7a8feacd6e0fc40af1c5d3d3573c1d1ac5" \
     -H 'Cookie: SWSId=1134c7bfcfa34d5c1015dfd473ab0cfa;'
```

Response:
```json
{"discoveries":{"discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68":true},"transports":null,"app_feedbacks":null,"version":"0.1.0","tag":"dev","os":"darwin"}
```

### Get Node Message
#### Usage
```
URI: /node/getMsg
Method: TBA
```

Example:
```sh
```

Response:
```json
```
	
### Get Node Applications
Retrieves a list of the Skywire applications that are currently active and running on the Node.

The response provided will be an array of running applications, specifying the application `key` and `attribure` which defines to the type of application.

#### Usage
```
URI: /node/getApps
Method: Get
```

Example:
```sh
curl "http://127.0.0.1:6001/node/getApps?token=261f61d536c89ecb0e51a31c1a438a278e298e61297dab9afa20199f264bf41c" \
     -H 'Cookie: SWSId=12384f4a4e2c60c160bdc190d0b1f331'
```

Response:
```json
[{"key":"01c8d1cfb7167371ce2ba8fd7c7341bca0c2a511052650164bcb368386232617ac","attributes":["sockss"],"allow_nodes":null}]
```

### Reboot Node
Reboots (restarts) the Node application. 
An example usage of this API can be found in the Manager Web UI.

#### Usage
```
URI: /node/reboot
Method: TBA
```

Example:
```sh
```

Response:
```
```

## Run

### Run SSHS
Runs (starts) the Skywire `sshs` (SSH Server) application on the Node.

#### Usage
```
URI: /node/run/sshs
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Run SSHC
Runs (starts) the Skywire `sshc` (SSH Client) application on the Node.

#### Usage
```
URI: /node/run/sshc
Method: TBA
```

Example:
```sh
```

Response:
```
```
	
### Run SOCKSS
Runs (starts) the Skywire `sockss` (Socks Server) application on the Node.

#### Usage
```
URI: /node/run/sockss
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Run SOCKSC
Runs (starts) the Skywire `socksc` (Socks Client) application on the Node.

#### Usage
```
URI: /node/run/socksc
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Run UPDATE
Runs (starts) the Skywire software update process. Use `/node/run/checkUpdate` to check if a new software version is avaiable first.

#### Usage
```
URI: /node/run/update
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Run Check Update
Runs (starts) the Skywire software update check. Used to check if a new version of Skywire is available. Use `/node/run/update` to perform the update.

#### Usage
```
URI: /node/run/checkUpdate
Method: TBA
```

Example:
```sh
```

Response:
```
```
	
### Set Node Config
#### Usage
```
URI: /node/run/setNodeConfig
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Update Node
#### Usage
```
URI: /node/run/updateNode
Method: TBA
```

Example:
```sh
```

Response:
```
```
	
### Run Shell
#### Usage
```
URI: /node/run/runShell
Method: TBA
```

Example:
```sh
```

Response:
```
```
	
### Run Command
#### Usage
```
URI: /node/run/runCmd
Method: TBA
```

Example:
```sh
```

Response:
```
```
	
### Get Shell Output
#### Usage
```
URI: /node/run/getShellOutput
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Serach Services
#### Usage
```
URI: /node/run/searchServices
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Search Services Results
#### Usage
```
URI: /node/run/searchServicesResults
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Get Auto Start Config
#### Usage
```
URI: /node/run/getAutoStartConfig
Method: TBA
```

Example:
```sh
```

Response:
```
```
TBA

### Set Auto Start Config
#### Usage
```
URI: /node/run/setAutoStartConfig
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Close Application
#### Usage
```
URI: /node/run/closeApp
Method: TBA
```

Example:
```sh
```

Response:
```
```

### Run TERM
#### Usage
```
URI: /node/run/term
Method: Get
```

Example:
```sh
```

Response:
```
```