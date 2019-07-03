#!/usr/bin/env bash
curl --data  {'"recipient":"'$PK_A'", "message":"Hello Joe!"}' -X POST  $CHAT_C
curl --data  {'"recipient":"'$PK_C'", "message":"Hello Mike!"}' -X POST  $CHAT_A
