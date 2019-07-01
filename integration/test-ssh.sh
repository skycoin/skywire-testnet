#!/usr/bin/env bash

echo Starting ssh test
echo Press Ctrl-C to exit

source ./integration/ssh/env-vars.sh

export N=1
for i in {1..16}
do 
	echo Test with $N lines
	mkdir -p ./logs/ssh/$N

	echo Killing visors and SSH-cli
	echo Killing $(ps aux |grep "[V]isorA\|[V]isorB\|[V]isorC\|[s]kywire/SSH-cli" |awk '{print $2}')
	kill $(ps aux |grep "[V]isorA\|[V]isorB\|[V]isorC\|[s]kywire/SSH-cli" |awk '{print $2}')

	echo Restarting visors
	./bin/visor ./integration/ssh/visorA.json --tag VisorA &> ./logs/ssh/$N/visorA.log &
	./bin/visor ./integration/intermediary-visorB.json --tag VisorB  &> ./logs/ssh/$N/visorB.log &
	./bin/visor ./integration/ssh/visorC.json --tag VisorC &> ./logs/ssh/$N/visorC.log &

	sleep 20
	echo Trying SSH-cli
	export CMD=$(echo ./bin/SSH-cli $PK_A \"loop -n $N echo A\")
	echo $CMD 
	eval $CMD &>./logs/ssh/$N/SSH-cli.out 


	export N=$(($N*2))
done
