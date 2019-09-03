#!/usr/bin/env bash

export GO111MODULE=on
export COMPOSE_FILE=docker-compose.yml:docker-compose.nodes.yml
set -e -x

# Store current path
export SWPATH=$(pwd)

# Clone from master
rm -rf /tmp/skywire-services &> /dev/null
cd /tmp
git clone https://$GITHUB_TOKEN:x-oauth-basic@github.com/watercompany/skywire-services.git  --branch master --depth 1
# git clone git@github.com:watercompany/skywire-services.git --branch master --depth 1
cd skywire-services

# go mod edit
go mod edit  -replace=github.com/skycoin/skywire@mainnet=$SWPATH

# Checking build 
make dep
make build

# Running regular tests
make test

# Checking e2e-build
make e2e-clean
make e2e-build
make e2e-run

# Running e2e tests
make e2e-test
