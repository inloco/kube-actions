ARG DOCKER_VERSION

FROM golang:1.20.10-alpine3.17 AS build
RUN CGO_ENABLED=0 go install -v github.com/go-delve/delve/cmd/dlv@latest
WORKDIR /go/src/github.com/inloco/kube-actions/dind
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM docker:${DOCKER_VERSION}-dind-rootless
USER root
RUN apk add --no-cache jq socat fuse3 sudo shadow && \
    export CRUN_URL=https://github.com/containers/crun/releases/download/1.7.2/crun-1.7.2-linux-amd64 && \
    wget -qO /usr/local/bin/crun ${CRUN_URL} && \
    chmod +x /usr/local/bin/crun && \
    export SLIRP4NETNS_URL=$(wget -qO- https://api.github.com/repos/rootless-containers/slirp4netns/releases | jq -r '.[0].assets[] | select(.name | match("x86_64$")) | .browser_download_url') && \
    wget -qO /usr/local/bin/slirp4netns ${SLIRP4NETNS_URL} && \
    chmod +x /usr/local/bin/slirp4netns && \
    mkdir /opt/containerd && \
    chown rootless:rootless /opt/containerd && \
    mkdir /run/netns && \
    chown rootless:rootless /run/netns && \
    sed -i s/0.0.0.0/127.0.0.1/g $(which dockerd-entrypoint.sh) && \
    unlink /sbin/init && \
    wget https://github.com/containers/fuse-overlayfs/releases/download/v1.13/fuse-overlayfs-x86_64 -O fuse-overlayfs && \
    chmod +x fuse-overlayfs && \
    cp fuse-overlayfs /usr/local/bin/fuse-overlayfs

# Sounds a bit off to add root permission to rootless user, but its privileges are removed as soon as the container starts
RUN echo '%wheel ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/wheel && \
    adduser rootless wheel && \
    mkdir /run/user/$(id -u rootless) && \
    chown -R rootless:rootless /home/rootless /run/user/$(id -u rootless)/

COPY --from=build /go/bin/dlv /usr/local/bin/dlv
COPY --from=build /go/bin/dind /sbin/init
USER rootless
ENV DOCKER_HOST=tcp://127.0.0.1:2375
ENV DOCKER_TLS_CERTDIR=
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_NET=slirp4netns
ENV DOCKERD_ROOTLESS_ROOTLESSKIT_MTU=65520
ENV DOCKERD_ENTRYPOINT_ARGS="--storage-driver fuse-overlayfs --add-runtime crun=/usr/local/bin/crun --default-runtime crun --experimental --registry-mirror https://mirror.gcr.io --tls=false"
ENTRYPOINT ["sudo", "-E", "/sbin/init"]
