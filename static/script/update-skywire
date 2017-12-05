#!/bin/bash

GOBIN_DIR=/usr/local/skywire-go

echo "Updating SkyWire..."
cd ${GOBIN_DIR}/src/github.com/skycoin/skywire
git reset --hard
git pull origin master
cd ${GOBIN_DIR}/src/github.com/skycoin/skywire/cmd
# [[ -d ${GOBIN_DIR}/pkg/linux_arm64/github.com/skycoin ]] && rm -rf ${GOBIN_DIR}/pkg/linux_arm64/github.com/skycoin
go install ./... 2>> /tmp/skywire_install_errors.log

echo "Updating SkyWire Script..."
cd /usr/local/skywire-script
git reset --hard
git pull origin master

echo "Updating SkyWire Web..."
cd /usr/local/skywire-static
[[ -d skywire-manager ]] && cd skywire-manager
git reset --hard
git pull origin master

echo "Updating SkyWire Screen..."
cd /usr/local/skywire-script
cp -f screen/10-header /etc/update-motd.d/
cp -f screen/99-point-to-faq /etc/update-motd.d/

echo "Kill SkyWire Process..."
[[ -f /tmp/skywire-pids/manager.pid ]] && pkill -F /tmp/skywire-pids/manager.pid && rm -rf /tmp/skywire-pids/manager.pid
[[ -f /tmp/skywire-pids/node.pid ]] && pkill -F /tmp/skywire-pids/node.pid && rm -rf /tmp/skywire-pids/node.pid

echo "Rebooting..."
reboot