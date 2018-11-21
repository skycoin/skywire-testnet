#!/bin/bash

# This script will be copied to each node con /tmp
# and will be run at the end to upgrade it
# 

# vars we will need
HOME=/root
GOPATH=/usr/local/skywire/go
PATH="/usr/local/go/bin:/usr/local/skywire/go/bin:${PATH}"
SKYCOIN_DIR=${GOPATH}/src/github.com/skycoin
SKYWIRE_DIR=${SKYCOIN_DIR}/skywire
SKYWIRE_SCRIPTS=${SKYWIRE_DIR}/static/script/
SKYWIRE_GIT_URL="https://github.com/skycoin/skywire.git"

export HOME
export GOPATH

# change to skywire repo path
cd ${SKYWIRE_DIR}

# set remote URL for git
git reset --hard
git remote set-url origin ${SKYWIRE_GIT_URL}

# TODO activate this on production
#git pull || git pull

# TODO remove this on production
#######################################
cd ..
rm -rdf skywire
tar -xf skywire-new.tar
cd skywire
#######################################

# remove old go bins to force a total rebuild
cd ${GOPATH=/usr/local/skywire/go}
rm -rdf bin

# copy defaults
cp ${SKYWIRE_SCRIPTS}/skywire.defaults /etc/default/skywire

# compile
cd ${SKYWIRE_DIR}/cmd
go install -v ./... &>1 

# restarting services
systemctl restart skywire-manager
systemctl restart skywire-node

# reset root password
echo "root:skywire" | chpasswd

# Done
