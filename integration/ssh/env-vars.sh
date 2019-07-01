# This script needs to be `source`d from bash-compatible shell
# E.g. `source ./integration/ssh/env-vars.sh` or `. ./integration/ssh/env-vars.sh`
export PK_A=$(jq -r ".visor.static_public_key" ./integration/ssh/visorA.json)
export RPC_A=$(jq -r ".interfaces.rpc" ./integration/ssh/visorA.json)
export PK_B=$(jq -r ".visor.static_public_key" ./integration/intermediary-visorB.json)
export RPC_B=$(jq -r ".interfaces.rpc" ./integration/intermediary-visorB.json)
export PK_C=$(jq -r ".visor.static_public_key" ./integration/ssh/visorC.json)
export RPC_C=$(jq -r ".interfaces.rpc" ./integration/ssh/visorC.json)

alias CLI_A='./skywire-cli --rpc $RPC_A'
alias CLI_B='./skywire-cli --rpc $RPC_B'
alias CLI_C='./skywire-cli --rpc $RPC_C'

export MSGD=https://messaging.discovery.skywire.skycoin.net
export TRD=https://transport.discovery.skywire.skycoin.net
export RF=https://routefinder.skywire.skycoin.net

alias RUN_A='go run ./cmd/visor ./integration/messaging/visorA.json --tag VisorA'
alias RUN_B='go run ./cmd/visor ./integration/intermediary-visorB.json --tag VisorB'
alias RUN_C='go run ./cmd/visor ./integration/messaging/visorC.json --tag VisorC'

echo PK_A: $PK_A
echo PK_B: $PK_B
echo PK_C: $PK_C
