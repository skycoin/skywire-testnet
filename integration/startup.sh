#!/usr/bin/env bash

# Use this script:
# - inside tmux session created by run-*-env.sh scripts 
# - or standalone `source ./integration/[name of environment]/env-vars.sh && ./integration/startup.sh` 

./skywire-cli --rpc $RPC_A visor add-tp $PK_B
./skywire-cli --rpc $RPC_C visor add-tp $PK_B
sleep 1

echo "VisorA Transports:"
./skywire-cli --rpc $RPC_A visor ls-tp

echo "VisorB Transports:"
./skywire-cli --rpc $RPC_B visor ls-tp
