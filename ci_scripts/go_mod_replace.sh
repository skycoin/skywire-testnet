#!/usr/bin/env bash

if [[ $# -ne 2 ]]; then
    exit 0
fi

# Inputs.
action=$1
filename=$2

# Line to comment/uncomment in go.mod
line="replace github.com\/SkycoinProject\/dmsg => ..\/dmsg"

function print_usage() {
    echo $"Usage: $0 (comment|uncomment) <filename>"
}

case "$action" in
    comment)
        echo "commenting ${filename}..."
        sed -i -e "/$line/s/^\/*/\/\//" ${filename}
        ;;
    uncomment)
        echo "uncommenting ${filename}..."
        sed -i -e "/$line/s/^\/\/*//" ${filename}
        ;;
    help)
        print_usage
        ;;
    *)
        print_usage
        exit 1
        ;;
esac
