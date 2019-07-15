#!/usr/bin/env bash

mkdir -p ./logs
echo Press Ctrl-C to exit
for ((;;))
do 
	./bin/skywire-visor ./integration/intermediary-nodeB.json --tag NodeB 2>> ./logs/nodeB.log >> ./logs/nodeB.log &
	echo node starting NodeB
	sleep 25
	echo Killing NodeB on $(ps aux |grep "[N]odeB" |awk '{print $2}')
	kill $(ps aux |grep "[N]odeB" |awk '{print $2}')
	sleep 3
	echo Restarting NodeB
done
