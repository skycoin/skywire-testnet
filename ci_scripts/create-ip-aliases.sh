#!/usr/bin/env bash

if [[ "$OSTYPE" == "linux-gnu" ]]; then     
    for ((i=1; i<=255; i++)) 
    do 
        sudo ip addr add 12.12.12.$i/32 dev lo 
    done
elif [[ "$OSTYPE" == "darwin" ]]; then 
    for ((i=1; i<=255; i++))
    do 
        sudo ip addr add 12.12.12.$i/32 dev lo0
    done
fi