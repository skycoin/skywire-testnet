#!/usr/bin/env bash

## SKYWIRE

tmux new -s skywire -d

source ./integration/ssh/env-vars.sh

echo "Checking transport-discovery is up"
curl --retry 5  --retry-connrefused 1 --connect-timeout 5 https://transport.discovery.skywire.skycoin.net/security/nonces/$PK_A   

tmux rename-window -t skywire NodeA
tmux send-keys -t NodeA './skywire-node ./integration/ssh/nodeA.json --tag NodeA' C-m
tmux new-window -t skywire -n NodeB
tmux send-keys -t NodeB './skywire-node ./integration/intermediary-nodeB.json --tag NodeB' C-m
tmux new-window -t skywire -n NodeC
tmux send-keys -t NodeC './skywire-node ./integration/ssh/nodeC.json --tag NodeC' C-m

tmux new-window -t skywire -n shell

tmux send-keys -t shell 'source ./integration/ssh/env-vars.sh' C-m

tmux attach -t skywire
