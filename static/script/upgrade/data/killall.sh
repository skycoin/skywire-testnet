#!/bin/bash

# This scriot will be copied to each node con /tmp
# and will be used to killa any skywire app runnig
# 

# find manager
PID=`ps aux | grep manager | grep -v grep | grep 'web-dir' | awk '{print $2}'`
if [ "$PID" != "" ] ; then
    echo "Manager runnig, pid $PID, killing it"
    `which kill` -9 $PID
fi

# find manager
PID=`ps aux | grep node | grep -v grep | grep 'connect-manager' | awk '{print $2}'`
if [ "$PID" != "" ] ; then
    echo "Node runnig, pid $PID, killing it"
    `which kill` -9 $PID
fi
