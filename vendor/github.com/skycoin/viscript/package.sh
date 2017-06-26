#!/bin/bash

# Set verbosity variable to false
let v=false

# Check if -v is passed and if true set verbosity to true to print everything
if [ "$1" = "-v" ]; then
    v=true
fi

# Define print verbose function that looks at the verbosity option and echoes
pv () {
    if [ $v = true ]; then
        echo "[ • ]" $1
    fi
}

# Get executable program name depending on running shell environment
getPlatformExeName () {
    local platformExeName=$1;

    if [ "$(expr substr $(uname -s) 1 10)" == "MINGW32_NT" ]; then
        echo "$platformExeName.exe";
    elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW64_NT" ]; then
        echo "$platformExeName.exe";
    else
        echo "$platformExeName";
    fi
}

skycoinName=$(getPlatformExeName "skycoin")
echo $skycoinName

echo "Start of packaging viscript and binaries:"

# Set Root directory name
readonly ROOT_DIR="$PWD/Viscript"
pv "Setting root directory to $ROOT_DIR"

# Make directory for Viscript
if [ ! -d $ROOT_DIR ]; then
    mkdir $ROOT_DIR
    pv "Creating root directory"
else
    pv "Root directory already exists, cleaning it up."
    rm -rf $ROOT_DIR 
    mkdir $ROOT_DIR
fi

# Make bin folder inside of ROOT_DIR
pv "Creating bin in root directory"
mkdir $ROOT_DIR/bin

# Make skycoin folder inside of ROOT_DIR/bin
pv "Creating skycoin directory inside root/bin"
mkdir $ROOT_DIR/bin/skycoin

# Make meshnet folder inside of ROOT_DIR/bin
pv "Creating meshnet dir inside root/bin"
mkdir $ROOT_DIR/bin/meshnet

# Set the Skywire path from github
githubSkywirePath="github.com/skycoin/skywire"

# Go get skywire
pv "Go getting Skywire: $githubSkywirePath"
go get -u -d $githubSkywirePath &>/dev/null

# Set local skywire path
localSkywirePath="$GOPATH/src/$githubSkywirePath"
pv "Local Skywire path set to: $localSkywirePath"

# Check if skywire directory exists in gopath
if [ ! -d "$localSkywirePath" ]; then
    pv "Skywire directory doesn't exist in $GOPATH/src/github.com/"
    exit 1
else
    pv "Skywire directory verified" 
fi

# Set the Skycoin path from github
githubSkycoinPath="github.com/skycoin/skycoin"

# Go get skycoin
pv "Go getting Skycoin: $githubSkycoinPath"
go get -u -d $githubSkycoinPath &>/dev/null

# Set local skycoin path
localSkycoinPath="$GOPATH/src/$githubSkycoinPath"
pv "Local Skycoin path set to: $localSkycoinPath"

# Check if skycoin directory exists in gopath
if [ ! -d "$localSkycoinPath" ]; then
    pv "Skycoin directory doesn't exist in $GOPATH/src/github.com/"
    exit 1
else
    pv "Skycoin directory verified" 
fi

# Change Directory to local skycoin path
pv "Changing directory to Skycoin main file"
cd "$localSkycoinPath/cmd/skycoin/"

# Get all dependencies for skycoin.go
pv "Getting all dependencies for Skycoin"
go get -d ./...

# Get Skycoin exe platform independent binary name
skycoinExeName=$(getPlatformExeName "skycoin")

# Build skycoin.go 
pv "Building Skycoin binary"
go build -o $skycoinExeName skycoin.go

# Check if skycoin binary was built successfully
if [ ! -f "$skycoinExeName" ]; then
    pv "Building Skycoin binary failed. Exiting"
    exit 1
fi

# Create skycoin directory inside the root/bin 
pv "Copying skycoin binary inside $ROOT_DIR/bin/skycoin/"
mv $skycoinExeName $ROOT_DIR/bin/skycoin/

# Change to src/gui/static 
pv "Changing directory to src/gui"
cd "$localSkycoinPath/src/gui/"

# Make directory for skycoin statics
pv "Making directory static inside local skycoin/"
mkdir "$ROOT_DIR/bin/skycoin/static/"

# Copy static folder to the local skycoin path inside newly created static/
pv "Copying static for Skycoin"
cp -R "static/" "$ROOT_DIR/bin/skycoin/"

# Change directory to local Skycoin cli 
pv "Changing directory to Skycoin cli"
cd "$localSkycoinPath/cmd/cli/"

# Get all dependencies for cli.go
pv "Getting all dependencies for Skycoin cli"
go get -d ./...

# Get skycoin cli platform independent binary name
skycoinCliExeName=$(getPlatformExeName "skycoin-cli")

# Build cli.go
pv "Building Skycoin cli"
go build -o $skycoinCliExeName cli.go

# Check if building cli was successfull
if [ ! -f "$skycoinCliExeName" ]; then
    pv "Building Skycoin cli failed. Exiting"
    exit 1
fi

# Move Skycoin cli to skycoin path
pv "Moving Skycoin cli to the root directory"
mv $skycoinCliExeName "$ROOT_DIR/bin/skycoin/"

# Change directory to local Skywire mesh server
pv "Changing directory to Skywire rpc server"
cd "$localSkywirePath/src/cmd/rpc/srv/"

# Get all dependencies for rpc_run.go
pv "Getting all dependencies for meshnet server"
go get -d ./...

# Get meshnet server platform independent name
meshnetServerExeName=$(getPlatformExeName "meshnet-server")

# Build run_rpc.go
pv "Building Skywire rpc server"
go build -o $meshnetServerExeName rpc_run.go

# Check if building cli was successfull
if [ ! -f "$meshnetServerExeName" ]; then
    pv "Building Skywire rpc server failed. Exiting"
    exit 1
fi

# Move Skywire mesh server to root dir
pv "Moving Skywire rpc server to the root directory"
mv $meshnetServerExeName "$ROOT_DIR/bin/meshnet/"

# Change directory to local Skywire mesh cli
pv "Changing directory to Skywire rpc mesh cli"
cd "$localSkywirePath/src/cmd/rpc/cli/"

# Get all dependencies for meshnet cli
pv "Getting all dependencies for meshnet cli"
go get -d ./...

# Get meshnet cli platform independent name
meshnetCliExeName=$(getPlatformExeName "meshnet-cli")

# Build cli.go
pv "Building Skywire rpc mesh cli"
go build -o $meshnetCliExeName cli.go

# Check if building cli was successfull
if [ ! -f "$meshnetCliExeName" ]; then
    pv "Building Skywire rpc mesh cli failed. Exiting"
    exit 1
fi

# Move Skywire mesh cli to root dir
pv "Moving Skywire rpc mesh cli to the root directory"
mv $meshnetCliExeName "$ROOT_DIR/bin/meshnet/"

# Change directory to root
pv "Changing directory to current working one"
cd "$ROOT_DIR/" && cd ..

# Get viscript platform independent name
viscriptExeName=$(getPlatformExeName "viscript")

# Build viscript.go
pv "Building viscript binary"
go build -o $viscriptExeName viscript.go

# Check if building viscript binary was successfull
if [ ! -f "$viscriptExeName" ]; then
    pv "Building viscript binary failed. Exiting"
    exit 1
fi

# Move viscript binary to the root
pv "Moving viscript binary into the root directory"
mv $viscriptExeName "$ROOT_DIR/"

# Copy README.md to the root
pv "Copying README.md for viscript"
cp "README.md" "$ROOT_DIR/"

# Move all assets that are required for viscript
pv "Copying assets folder inside the root directory for viscript"
cp -R "assets/" "$ROOT_DIR/"

# Get viscript cli platform independent binary name
viscriptCliExeName=$(getPlatformExeName "viscript-cli")

# Build viscript cli
pv "Building viscript cli"
go build -o $viscriptCliExeName rpc/cli/cli.go

# Check if building cli was successfull
if [ ! -f "$viscriptCliExeName" ]; then
    pv "Building viscript cli failed. Exiting"
    exit 1
fi

# Move viscript cli to the root directory 
pv "Moving viscript cli to the root directory"
mv $viscriptCliExeName "$ROOT_DIR/"

# Create viscript-cli.sh file that uses gotty to run the cli
pv "Creating bash file to run cli with gotty: https://github.com/yudai/gotty"
gottyCommand="gotty -w -p 9999 --reconnect ./viscript-cli"
echo $gottyCommand > "$ROOT_DIR/viscript-cli.sh" 

# Change directory to run_apptracker.go location
pv "Changing directory to apptracker creation script location"
cd "$PWD/mesh/run_mesh/apptracker"

# Get meshnet run apptracker platform independent binary name
meshnetAppTrackerExeName=$(getPlatformExeName "meshnet-run-apptracker")

# Build run_apptracker.go
pv "Building apptracker creation script"
go build -o $meshnetAppTrackerExeName run_apptracker.go

# Check if building apptracker creation script was successfull
if [ ! -f "$meshnetAppTrackerExeName" ]; then
    pv "Building apptracker creation script failed. Exiting"
    exit 1
fi

# Move apptracker creation script to root dir
pv "Moving apptracker creation script to the root directory"
mv $meshnetAppTrackerExeName "$ROOT_DIR/bin/meshnet/"

# Change directory to run_nm.go location
pv "Changing directory to nodemanager creation script location"
cd "../nodemanager"

# Get meshnet run nm platform independent binary name
meshnetRunNMExeName=$(getPlatformExeName "meshnet-run-nm")

# Build run_nm.go
pv "Building nodemanager creation script"
go build -o $meshnetRunNMExeName run_nm.go

# Check if building nodemanager creation script was successfull
if [ ! -f "$meshnetRunNMExeName" ]; then
    pv "Building nodemanager creation script failed. Exiting"
    exit 1
fi

# Move nodemanager creation script to root dir
pv "Moving nodemanager creation script to the root directory"
mv $meshnetRunNMExeName "$ROOT_DIR/bin/meshnet/"

# Change directory to run_node.go location
pv "Changing directory to node creation script location"
cd "../node"

# Get meshnet run node platform independent binary name
meshnetNodeExeName=$(getPlatformExeName "meshnet-run-node")

# Build run_node.go
pv "Building node creation script"
go build -o $meshnetNodeExeName run_node.go

# Check if building node creation script was successfull
if [ ! -f "$meshnetNodeExeName" ]; then
    pv "Building node creation script failed. Exiting"
    exit 1
fi

# Move node creation script to root dir
pv "Moving node creation script to the root directory"
mv $meshnetNodeExeName "$ROOT_DIR/bin/meshnet/"

# Change directory to run_vpn_client.go location
pv "Changing directory to vpn creation scripts location"
cd "../app/vpn"

if [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Get meshnet run vpn client platform independent binary name
    meshnetVpnClientExeName=$(getPlatformExeName "meshnet-run-vpn-client")

    # Build run_vpn_client.go
    pv "Building vpn client creation script"
    go build -o $meshnetVpnClientExeName run_vpn_client.go

    # Check if building vpn_client creation script was successfull
    if [ ! -f "$meshnetVpnClientExeName" ]; then
        pv "Building vpn client creation script failed. Exiting"
        exit 1
    fi

    # Move node creation script to root dir
    pv "Moving vpn client creation script to the root directory"
    mv $meshnetVpnClientExeName "$ROOT_DIR/bin/meshnet/"

    # Get meshnet run vpn server platform independent binary name
    meshnetVpnServerExeName=$(getPlatformExeName "meshnet-run-vpn-server")

    # Build run_vpn_server.go
    pv "Building vpn server creation script"
    go build -o $meshnetVpnServerExeName run_vpn_server.go

    # Check if building vpn_server creation script was successfull
    if [ ! -f "$meshnetVpnServerExeName" ]; then
        pv "Building vpn server creation script failed. Exiting"
        exit 1
    fi

    # Move node creation script to root dir
    pv "Moving vpn server creation script to the root directory"
    mv $meshnetVpnServerExeName "$ROOT_DIR/bin/meshnet/"

    # Change directory to run_socks_client.go location
    pv "Changing directory to socks creation scripts location"
    cd "../socks"

    # Get meshnet run socks client platform independent name
    meshnetSocksClientExeName=$(getPlatformExeName "meshnet-run-socks-client")

    # Build run_socks_client.go
    pv "Building socks client creation script"
    go build -o $meshnetSocksClientExeName run_socks_client.go

    # Check if building socks_client creation script was successfull
    if [ ! -f "$meshnetSocksClientExeName" ]; then
        pv "Building socks client creation script failed. Exiting"
        exit 1
    fi

    # Move node creation script to root dir
    pv "Moving socks client creation script to the root directory"
    mv $meshnetSocksClientExeName "$ROOT_DIR/bin/meshnet/"

    # Get meshnet socks server platform independent binary name
    meshnetSocksServerExeName=$(getPlatformExeName "meshnet-run-socks-server")

    # Build run_socks_server.go
    pv "Building socks server creation script"
    go build -o $meshnetSocksServerExeName run_socks_server.go

    # Check if building socks_server creation script was successfull
    if [ ! -f "$meshnetSocksServerExeName" ]; then
        pv "Building socks server creation script failed. Exiting"
        exit 1
    fi

    # Move node creation script to root dir
    pv "Moving socks server creation script to the root directory"
    mv $meshnetSocksServerExeName "$ROOT_DIR/bin/meshnet/"

fi

# TemporArily copy bin to root bin of the repo for testing
pv "CopyiNg generated bin directory to root bin of the repo for testing"
cd $ROOT_DIR/ && cd ..
cp -rf $ROOT_DIR/bin/ ./

# Copy appropriate platform dependent config file to for viscript to use
pv "Copying viscript os dependentent config yaml file"
if [ "$(expr substr $(uname -s) 1 5)" == "Linux" ] \
    || [ "$(expr substr $(uname -s) 1 6)" == "Darwin" ]; 
then
    cp "config.yaml" "$ROOT_DIR/config.yaml"
else
    cp "config-win.yaml" "$ROOT_DIR/config-win.yaml"
    mv "$ROOT_DIR/config-win.yaml" "$ROOT_DIR/config.yaml"
fi

# Print Done
pv "Done"

# TODO: zip here?
