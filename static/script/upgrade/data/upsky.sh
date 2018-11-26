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
git pull || git pull

# test, if the git failed then it must no have a new file
if [ ! -f "${SKYWIRE_SCRIPTS}/upgrade/README.md" ] ; then
    echo "ERROR: clone operation failed."
    exit
fi

# remove old go bins to force a total rebuild
cd ${GOPATH=/usr/local/skywire/go}
rm -rdf bin

# compile
cd ${SKYWIRE_DIR}/cmd
go install -v ./... &>1 

# reset root password
echo "root:skywire" | chpasswd

# Done
