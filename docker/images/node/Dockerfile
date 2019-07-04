ARG builder=node-builder
ARG base=alpine

FROM ${builder} as builder

FROM ${base} as base

COPY --from=builder /go/skywire/skywire-node skywire-node
COPY --from=builder /go/skywire/apps bin/apps
COPY --from=builder /go/skywire/docker/images/builder/update.sh update.sh

ENTRYPOINT [ "./skywire-node" ]
