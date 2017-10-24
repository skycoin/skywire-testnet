#!/usr/bin/env bash

source "./tool.sh"

sysOS=`uname -s`
if [ $sysOS == "Darwin" ];then
	inMac
elif [ $sysOS == "Linux" ];then
	inLinux
else
	echo "Other OS: $sysOS"
    exit 1
fi

install
rm -rf dist-manager
buildManager
