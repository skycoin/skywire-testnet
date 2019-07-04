#!/bin/sh

if type apt > /dev/null; then        
        apt update   
        apt upgrade -y
        apt install -y  ca-certificates iproute2   iputils-ping redis-server supervisor 

        # rm -rf /var/lib/apt/lists/* 
fi

if type apk > /dev/null; then    
        apk add --no-cache redis supervisor
fi
