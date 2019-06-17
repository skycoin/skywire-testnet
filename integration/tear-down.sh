#!/bin/sh

# In case skywire-nodes are not stopped properly.
kill $(ps aux |grep "[N]odeA" |awk '{print $2}')
kill $(ps aux |grep "[N]odeB" |awk '{print $2}')
kill $(ps aux |grep "[N]odeC" |awk '{print $2}')

echo Removing ./local
rm -rf ./local
