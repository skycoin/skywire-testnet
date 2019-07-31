#!/usr/bin/env bash

MSGD=messaging.discovery.skywire.skycoin.net
MSGD_GET="https://"$MSGD"/messaging-discovery/available_servers" 

echo -e "\nTCP delays. Measuring by ping:"
ping $MSGD -c 10 -q

if type mtr > /dev/null; then
    echo -e "\nTCP delays. Measuring by mtr:"
    mtr -y 2 --report  --report-cycles=5   $MSGD > /tmp/msgd-out.txt

    cat /tmp/msgd-out.txt
else
    echo -e "\nTCP delays. mtr not found. Install for detailed stats"
fi

if type vegeta > /dev/null; then
    echo -e "\nHTTP delays. Measuring by vegeta:"
    echo "GET "$MSGD_GET  \
        | vegeta attack -duration=10s |tee results.bin |vegeta report 
else
    echo -e "\nHTTP delays.vegeta not found\n. Install with \ngo get -u github.com/tsenart/vegeta\n for detailed stats"        
fi

echo -e "\nHTTP delays. Measuring by curl:"
curl $MSGD_GET >/dev/null
