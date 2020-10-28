ARG DOCKER_VERSION

FROM golang:alpine AS build
WORKDIR /build
ADD docker-iptables-glue docker-iptables-glue
RUN cd docker-iptables-glue && CGO_ENABLED=0 GOOS=linux go build

FROM docker:${DOCKER_VERSION}-dind-rootless
USER root

ENV CRUN_BASE_DOWNLOAD_PATH=https://github.com/containers/crun/releases/download \
    SLIRP3NETNS_BASE_DOWNLOAD_PATH=https://github.com/rootless-containers/slirp4netns/releases/download

RUN apk add --no-cache iproute2 jq && \
    export CRUN_LATEST=$(wget -qO- https://api.github.com/repos/containers/crun/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/crun "${CRUN_BASE_DOWNLOAD_PATH}/${CRUN_LATEST}/crun-${CRUN_LATEST}-linux-amd64" && \
    export SLIRP4NETNS_LATEST=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases/latest | jq -r .tag_name) && \
    wget -qO /usr/local/bin/slirp4netns "${SLIRP3NETNS_BASE_DOWNLOAD_PATH}/${SLIRP4NETNS_LATEST}/slirp4netns-x86_64" && \
    chmod +x /usr/local/bin/crun /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd /run/netns /run/rootless-docker && \
    chown rootless:rootless /opt/containerd /run/netns /run/rootless-docker && \
    chmod 700 /run/rootless-docker

COPY --from=build /build/docker-iptables-glue/docker-iptables-glue /sbin/
RUN chmod u+s /sbin/docker-iptables-glue
COPY ./init /sbin/

USER rootless
ENV DOCKER_HOST=unix:///run/rootless-docker/docker.sock \
    DOCKERD_ROOTLESS_ROOTLESSKIT_NET=slirp4netns \
    DOCKERD_ROOTLESS_ROOTLESSKIT_MTU=65520 \
    DOCKERD_ENTRYPOINT_ARGS="--ip 127.0.0.1 --add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror https://mirror.gcr.io"

ENTRYPOINT ["/sbin/init"]
