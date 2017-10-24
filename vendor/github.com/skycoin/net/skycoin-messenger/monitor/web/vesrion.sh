#!/usr/bin/env bash
function version_gt() { test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1"; }
function version_lt() { test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" != "$1"; }
compareVesrion() {
  vesrion=$(${1})
  if version_lt $vesrion ${2};then
    echo "Please upgrade ${3} version."
    exit 1
  fi
}



