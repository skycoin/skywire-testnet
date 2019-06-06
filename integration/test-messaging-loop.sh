#!/usr/bin/env bash
source ./integration/generic/env-vars.sh
echo "Press Ctrl-C to exit"
for  ((;;))
do
	curl --data  {'"recipient":"'$PK_A'", "message":"Hello Joe!"}' -X POST  $CHAT_C  
	curl --data  {'"recipient":"'$PK_C'", "message":"Hello Mike!"}' -X POST  $CHAT_A
done
