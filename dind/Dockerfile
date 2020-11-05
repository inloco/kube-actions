ARG DOCKER_VERSION

FROM golang:1.15-alpine AS build
WORKDIR /go/src/github.com/inloco/kube-actions/dind
COPY go.mod go.sum ./
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM docker:${DOCKER_VERSION}-dind-rootless
USER root

RUN export CRUN_BASE_DOWNLOAD_PATH=https://github.com/containers/crun/releases/download && \
    export SLIRP4NETNS_BASE_DOWNLOAD_PATH=https://github.com/rootless-containers/slirp4netns/releases/download && \
    apk add --no-cache socat jq && \
    export CRUN_LATEST=$(wget -qO- https://api.github.com/repos/containers/crun/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/crun "${CRUN_BASE_DOWNLOAD_PATH}/${CRUN_LATEST}/crun-${CRUN_LATEST}-linux-amd64" && \
    export SLIRP4NETNS_LATEST=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/slirp4netns "${SLIRP4NETNS_BASE_DOWNLOAD_PATH}/${SLIRP4NETNS_LATEST}/slirp4netns-x86_64" && \
    chmod +x /usr/local/bin/crun /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd /run/netns && \
    chown rootless:rootless /opt/containerd /run/netns

COPY ./init /sbin/
COPY --from=build /go/bin/dind /sbin/
RUN chmod u+s /sbin/dind

USER rootless
ENV DOCKER_HOST tcp://localhost:2375
ENV DOCKER_TLS_CERTDIR ""
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_NET slirp4netns
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_MTU 65520
ENV DOCKERD_ENTRYPOINT_ARGS "--add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror https://mirror.gcr.io"

ENTRYPOINT ["/sbin/init"]
