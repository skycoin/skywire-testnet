#!/usr/bin/env bash

echo Starting ssh test
echo Press Ctrl-C to exit

source ./integration/proxy/env-vars.sh

export N=1
for i in {1..16}
do 
	echo Test with $N requests
	mkdir -p ./logs/proxy/$N

	echo Killing visors
	echo Killing $(ps aux |grep "[V]isorA\|[V]isorB\|[V]isorC" |awk '{print $2}')
	kill $(ps aux |grep "[V]isorA\|[V]isorB\|[V]isorC" |awk '{print $2}')

	# This sleep needed to allow clean exit of visor
	sleep 10

	echo Restarting visorA and VisorB
	./bin/visor ./integration/proxy/visorA.json --tag VisorA &> ./logs/proxy/$N/visorA.log &
	./bin/visor ./integration/intermediary-visorB.json --tag VisorB  &> ./logs/proxy/$N/visorB.log &

	# TODO: improve this sleep
	sleep 5
	echo Restarting visorC
	./bin/visor ./integration/proxy/visorC.json --tag VisorC &> ./logs/proxy/$N/visorC.log &

	sleep 20
	echo Trying socks5 proxy

	for ((j=0; j<$N; j++))
	do 
		echo Request $j
		curl -v --retry 5 --retry-connrefused 1  --connect-timeout 5 -x socks5://123456:@localhost:9999 https://www.google.com &>> ./logs/proxy/$N/curl.out
	done

	export N=$(($N*2))
done
