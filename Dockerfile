FROM alpine:3.18
RUN apk add --no-cache lvm2
COPY lvminit /usr/local/bin/lvminit
ENTRYPOINT ["/usr/local/bin/lvminit"]
