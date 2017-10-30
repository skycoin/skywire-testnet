source "./vesrion.sh"

inMac() {
    if ! type "npm" > /dev/null; then
        echo "npm is not installed."
        exit 1c
    fi
    compareVesrion "npm -v" 4 "npm"
    if ! type "ng" > /dev/null; then
        echo "ng is not installed."
        exit 1
    fi
}
inLinux() {
    # Check if curl is installed.
    if ! type "npm" > /dev/null ; then
        echo "you can exec './linux_env.sh'"
        exit 1
    fi

    # Check if jq is installed.
    if ! type "ng" > /dev/null ; then
        echo "ng is not installed."
        exit 1
    fi
}
install() {
  cmd "npm install"
}
dev() {
  cmd "npm run start"
}

build(){
  cmd "npm run build-d"
}

buildManager(){
  cmd "npm run build-m"
}
cmd() {
    echo "[ RUNNING ] '${1}' ..."
    ${1}
    RETURN_VALUE=$?
    if [ ${RETURN_VALUE} -ne 0 ]; then
        err "command '${1}' failed with return value '${RETURN_VALUE}'"
        exit ${RETURN_VALUE}
    fi
}
