#!/usr/bin/env bash

## SKYWIRE

tmux new -s skywire -d

source ./integration/ssh/env-vars.sh

echo "Checking transport-discovery is up"
curl --retry 5  --retry-connrefused 1 --connect-timeout 5 https://transport.discovery.skywire.skycoin.net/security/nonces/$PK_A   

tmux rename-window -t skywire VisorA
tmux send-keys -t VisorA -l "./visor ./integration/ssh/visorA.json --tag VisorA $SYSLOG_OPTS"
tmux send-keys C-m
tmux new-window -t skywire -n VisorB
tmux send-keys -t VisorB -l "./visor ./integration/intermediary-visorB.json --tag VisorB $SYSLOG_OPTS"
tmux send-keys C-m
tmux new-window -t skywire -n VisorC
tmux send-keys -t VisorC -l "./visor ./integration/ssh/visorC.json --tag VisorC $SYSLOG_OPTS"
tmux send-keys C-m

tmux new-window -t skywire -n shell

tmux send-keys -t shell 'source ./integration/ssh/env-vars.sh' C-m

tmux attach -t skywire
