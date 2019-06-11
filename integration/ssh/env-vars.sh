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

echo PK_A: $PK_A
echo PK_B: $PK_B
echo PK_C: $PK_C
