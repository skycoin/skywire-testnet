#!/usr/bin/env bash

# Use this script:
# - inside tmux session created by run-*-env.sh scripts 
# - or standalone `source ./integration/[name of environment]/env-vars.sh && ./integration/startup.sh` 

./skywire-cli --rpc $RPC_A node add-tp $PK_B
./skywire-cli --rpc $RPC_C node add-tp $PK_B
sleep 1

echo "NodeA Transports:"
./skywire-cli --rpc $RPC_A node ls-tp

echo "NodeB Transports:"
./skywire-cli --rpc $RPC_B node ls-tp
