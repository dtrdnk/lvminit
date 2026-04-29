FROM mirror.gcr.io/alpine:3.22.4
RUN apk add --no-cache lvm2
COPY lvminit /usr/local/bin/lvminit
ENTRYPOINT ["/usr/local/bin/lvminit"]
