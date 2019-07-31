# intended to be sourced  `source ./integration/tcp-tr/env-vars.sh`

export RPC_A=192.168.1.2:3435
export RPC_C=192.168.1.3:3435

alias CLI_A='./skywire-cli --rpc $RPC_A'
alias CLI_C='./skywire-cli --rpc $RPC_C'

export PK_A=$(./skywire-cli --rpc $RPC_A node pk)
export PK_C=$(./skywire-cli --rpc $RPC_C node pk)

export CHAT_A=http://192.168.1.2:8001/message
export CHAT_C=http://192.168.1.3:8001/message