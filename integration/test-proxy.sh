#!/usr/bin/env bash

echo Starting ssh test
echo Press Ctrl-C to exit

source ./integration/proxy/env-vars.sh

export N=1
for i in {1..16}
do 
	echo Test with $N requests
	mkdir -p ./logs/proxy/$N

	echo Killing nodes 
	echo Killing $(ps aux |grep "[N]odeA\|[N]odeB\|[N]odeC" |awk '{print $2}')
	kill $(ps aux |grep "[N]odeA\|[N]odeB\|[N]odeC" |awk '{print $2}')

	# This sleep needed to allow clean exit of node
	sleep 10

	echo Restarting nodeA and NodeB
	./bin/skywire-node ./integration/proxy/nodeA.json --tag NodeA &> ./logs/proxy/$N/nodeA.log &
	./bin/skywire-node ./integration/intermediary-nodeB.json --tag NodeB  &> ./logs/proxy/$N/nodeB.log &

	# TODO: improve this sleep
	sleep 5
	echo Restarting nodeC
	./bin/skywire-node ./integration/proxy/nodeC.json --tag NodeC &> ./logs/proxy/$N/nodeC.log &

	sleep 20
	echo Trying socks5 proxy

	for ((j=0; j<$N; j++))
	do 
		echo Request $j
		curl -v --retry 5 --retry-connrefused 1  --connect-timeout 5 -x socks5://123456:@localhost:9999 https://www.google.com &>> ./logs/proxy/$N/curl.out
	done

	export N=$(($N*2))
done