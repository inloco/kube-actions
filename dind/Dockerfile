ARG DOCKER_VERSION

FROM golang:1.16.0-alpine AS build
WORKDIR /go/src/github.com/inloco/kube-actions/dind
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM docker:${DOCKER_VERSION}-dind-rootless
USER root
RUN apk add --no-cache jq socat && \
    export CRUN_URL=$(wget -qO- https://api.github.com/repos/containers/crun/releases | jq -r '.[0].assets[] | select(.name | match("linux-amd64$")) | .browser_download_url') && \
    wget -qO /usr/local/bin/crun ${CRUN_URL} && \
    chmod +x /usr/local/bin/crun && \
    export SLIRP4NETNS_URL=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases | jq -r '.[0].assets[] | select(.name | match("x86_64$")) | .browser_download_url') && \
    wget -qO /usr/local/bin/slirp4netns ${SLIRP4NETNS_URL} && \
    chmod +x /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd && \
    chown rootless:rootless /opt/containerd && \
    mkdir /run/netns && \
    chown rootless:rootless /run/netns && \
    unlink /sbin/init
COPY --from=build /go/bin/dind /sbin/init
RUN chmod +s /sbin/init
USER rootless
ENV DOCKER_HOST tcp://localhost:2375
ENV DOCKER_TLS_CERTDIR ""
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_NET slirp4netns
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_MTU 65520
ENV DOCKERD_ENTRYPOINT_ARGS --add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror https://mirror.gcr.io
ENTRYPOINT ["/sbin/init"]
