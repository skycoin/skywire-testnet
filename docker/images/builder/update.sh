#!/bin/sh

if type apt > /dev/null; then
        apt-get update && apt-get install -y --no-install-recommends \
                ca-certificates \
        && rm -rf /var/lib/apt/lists/* 
fi

if type apk > /dev/null; then
        
        apk update 
        apk upgrade 
        apk add --no-cache ca-certificates openssl
        update-ca-certificates --fresh
fi
