# skywire build binaries
# reference https://github.com/skycoin/skywire
ARG IMAGE_FROM=busybox:1.29-glibc
FROM golang:1.10-stretch AS build-go
ARG ARCH=amd64
ARG GOARM
ARG CC=gcc

COPY . $GOPATH/src/github.com/skycoin/skywire

RUN apt-get update \
    && apt-get -y install build-essential crossbuild-essential-armhf crossbuild-essential-arm64 automake gcc-arm-linux-gnueabihf

RUN cd $GOPATH/src/github.com/skycoin/skywire && \
    GOARCH=$ARCH GOARM=$GOARM GOOS=linux CGO_ENABLED=1 CC=$CC \
    go install -a -installsuffix cgo ./... && \
    sh -c "if test -d $GOPATH/bin/linux_arm ; then mv $GOPATH/bin/linux_arm/* $GOPATH/bin/; fi; \
           if test -d $GOPATH/bin/linux_arm64 ; then mv $GOPATH/bin/linux_arm64/* $GOPATH/bin/; fi"


# skywire manager assets
FROM node:8.9 AS build-node

# `unsafe` flag used as work around to prevent infinite loop in Docker
# see https://github.com/nodejs/node-gyp/issues/1236
RUN npm install -g --unsafe @angular/cli && \
    git clone https://github.com/skycoin/net.git /home/node/net && \
    cd /home/node/net/skycoin-messenger/monitor/web && \
    ./build-manager.sh


# skywire image
FROM $IMAGE_FROM

ENV DATA_DIR=/root/.skywire

#RUN adduser -D skywire

#USER skywire

# copy binaries and asset
COPY --from=build-go /go/bin/* /bin/
COPY --from=build-go /go/bin/sockss .
COPY --from=build-node /home/node/net/skycoin-messenger/monitor/web/dist-manager /usr/local/skycoin/net/skycoin-messenger/monitor/web/dist-manager

VOLUME $DATA_DIR

EXPOSE 5000 5998 8000 6001

# start manager
CMD ["manager", "-web-dir", "/usr/local/skycoin/net/skycoin-messenger/monitor/web/dist-manager"]
