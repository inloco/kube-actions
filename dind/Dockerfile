ARG DOCKER_VERSION

FROM golang:alpine AS build
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build

FROM docker:${DOCKER_VERSION}-dind-rootless
USER root

ENV CRUN_BASE_DOWNLOAD_PATH=https://github.com/containers/crun/releases/download \
    SLIRP4NETNS_BASE_DOWNLOAD_PATH=https://github.com/rootless-containers/slirp4netns/releases/download

RUN apk add --no-cache iproute2 socat jq && \
    export CRUN_LATEST=$(wget -qO- https://api.github.com/repos/containers/crun/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/crun "${CRUN_BASE_DOWNLOAD_PATH}/${CRUN_LATEST}/crun-${CRUN_LATEST}-linux-amd64" && \
    export SLIRP4NETNS_LATEST=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/slirp4netns "${SLIRP4NETNS_BASE_DOWNLOAD_PATH}/${SLIRP4NETNS_LATEST}/slirp4netns-x86_64" && \
    chmod +x /usr/local/bin/crun /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd /run/netns && \
    chown rootless:rootless /opt/containerd /run/netns

COPY ./init /sbin/
COPY --from=build /build/dind /sbin/
RUN chmod u+s /sbin/dind

USER rootless
ENV DOCKER_HOST=tcp://localhost:2375 \
    DOCKERD_ROOTLESS_ROOTLESSKIT_NET=slirp4netns \
    DOCKERD_ROOTLESS_ROOTLESSKIT_MTU=65520 \
    DOCKERD_ENTRYPOINT_ARGS="--add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror https://mirror.gcr.io"

ENTRYPOINT ["/sbin/init"]
