#!/bin/sh

# In case visors are not stopped properly.
kill $(ps aux |grep "[V]isorA" |awk '{print $2}')
kill $(ps aux |grep "[V]isorB" |awk '{print $2}')
kill $(ps aux |grep "[V]isorC" |awk '{print $2}')

echo Removing ./local
rm -rf ./local
