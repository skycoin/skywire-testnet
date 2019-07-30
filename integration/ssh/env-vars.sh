# This script needs to be `source`d from bash-compatible shell
# E.g. `source ./integration/ssh/env-vars.sh` or `. ./integration/ssh/env-vars.sh`
export PK_A=$(jq -r ".node.static_public_key" ./integration/ssh/nodeA.json)
export RPC_A=$(jq -r ".interfaces.rpc" ./integration/ssh/nodeA.json)
export PK_B=$(jq -r ".node.static_public_key" ./integration/intermediary-nodeB.json)
export RPC_B=$(jq -r ".interfaces.rpc" ./integration/intermediary-nodeB.json)
export PK_C=$(jq -r ".node.static_public_key" ./integration/ssh/nodeC.json)
export RPC_C=$(jq -r ".interfaces.rpc" ./integration/ssh/nodeC.json)

alias CLI_A='./skywire-cli --rpc $RPC_A'
alias CLI_B='./skywire-cli --rpc $RPC_B'
alias CLI_C='./skywire-cli --rpc $RPC_C'

export DMSGD=https://dmsg.discovery.skywire.skycoin.net
export TRD=https://transport.discovery.skywire.skycoin.net
export RF=https://routefinder.skywire.skycoin.net

alias RUN_A='go run ./cmd/skywire-visor ./integration/messaging/nodeA.json --tag NodeA'
alias RUN_B='go run ./cmd/skywire-visor ./integration/intermediary-nodeB.json --tag NodeB'
alias RUN_C='go run ./cmd/skywire-visor ./integration/messaging/nodeC.json --tag NodeC'

echo PK_A: $PK_A
echo PK_B: $PK_B
echo PK_C: $PK_C
