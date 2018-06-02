# skywire build binaries
# reference https://github.com/skycoin/skywire
ARG IMAGE_FROM=alpine:3.7
FROM golang:1.9-alpine AS build-go
ARG ARCH=amd64
ARG GOARM

COPY . $GOPATH/src/github.com/skycoin/skywire

RUN cd $GOPATH/src/github.com/skycoin/skywire && \
    GOARCH=$ARCH GOARM=$GOARM CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo ./... && \
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

# copy binaries and assets
COPY --from=build-go /go/bin/* /usr/bin/
COPY --from=build-go /go/bin/sockss .
COPY --from=build-node /home/node/net/skycoin-messenger/monitor/web/dist-manager /usr/local/skycoin/net/skycoin-messenger/monitor/web/dist-manager

VOLUME $DATA_DIR

EXPOSE 5000 5998 8000 6001

# start manager
CMD ["manager", "-web-dir", "/usr/local/skycoin/net/skycoin-messenger/monitor/web/dist-manager"]
