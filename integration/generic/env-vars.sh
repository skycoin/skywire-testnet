# This script needs to be `source`d from bash-compatible shell
# E.g. `source ./integration/generic/env-vars.sh` or `. ./integration/generic/env-vars.sh`
export PK_A=$(jq -r ".node.static_public_key" ./integration/generic/nodeA.json)
export RPC_A=$(jq -r ".interfaces.rpc" ./integration/generic/nodeA.json)
export PK_B=$(jq -r ".node.static_public_key" ./integration/intermediary-nodeB.json)
export RPC_B=$(jq -r ".interfaces.rpc" ./integration/intermediary-nodeB.json)
export PK_C=$(jq -r ".node.static_public_key" ./integration/generic/nodeC.json)
export RPC_C=$(jq -r ".interfaces.rpc" ./integration/generic/nodeC.json)

export CHAT_A=http://localhost:8000/message
export CHAT_C=http://localhost$(jq -r '.apps [] |select(.app=="skychat")| .args[1] ' ./integration/generic/nodeC.json)/message

alias CLI_A='./skywire-cli --rpc $RPC_A'
alias CLI_B='./skywire-cli --rpc $RPC_B'
alias CLI_C='./skywire-cli --rpc $RPC_C'

export MSGD=https://messaging.discovery.skywire.skycoin.net
export TRD=https://transport.discovery.skywire.skycoin.net
export RF=https://routefinder.skywire.skycoin.net

echo PK_A: $PK_A
echo PK_B: $PK_B
echo PK_C: $PK_C

echo CHAT_A: $CHAT_A
echo CHAT_C: $CHAT_C
