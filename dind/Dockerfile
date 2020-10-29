ARG DOCKER_VERSION

FROM docker:${DOCKER_VERSION}-dind-rootless

USER root

RUN export CRUN_LATEST=$(wget -qO- https://api.github.com/repos/containers/crun/releases/latest | sed -En 's/.*"tag_name"\s*:\s*"((\\"|[^"])*)".*/\1/p') && \
    wget -qO /usr/local/bin/crun "https://github.com/containers/crun/releases/download/${CRUN_LATEST}/crun-${CRUN_LATEST}-linux-amd64" && \
    chmod +x /usr/local/bin/crun && \
    export SLIRP4NETNS_LATEST=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases/latest | sed -En 's/.*"tag_name"\s*:\s*"((\\"|[^"])*)".*/\1/p') && \
    wget -qO /usr/local/bin/slirp4netns "https://github.com/rootless-containers/slirp4netns/releases/download/${SLIRP4NETNS_LATEST}/slirp4netns-x86_64" && \
    chmod +x /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd && \
    chown rootless:rootless /opt/containerd && \
    mkdir /run/netns && \
    chown rootless:rootless /run/netns

COPY ./init /sbin/init

USER rootless

ENV DOCKERD_ROOTLESS_ROOTLESSKIT_NET slirp4netns
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_MTU 65520
ENV DOCKERD_ENTRYPOINT_ARGS --add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror=https://mirror.gcr.io

ENTRYPOINT ["/sbin/init"]
