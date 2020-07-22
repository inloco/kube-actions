ARG DOCKER_VERSION=19.03

FROM docker:${DOCKER_VERSION}-dind-rootless

USER root

RUN export CRUN_LATEST=$(wget -qO- https://api.github.com/repos/containers/crun/releases/latest | sed -En 's/.*"tag_name"\s*:\s*"((\\"|[^"])*)".*/\1/p') && \
    wget -qO /usr/local/bin/crun "https://github.com/containers/crun/releases/download/${CRUN_LATEST}/crun-${CRUN_LATEST}-static-x86_64" && \
    chmod +x /usr/local/bin/crun

USER rootless

COPY ./init /init

ENTRYPOINT [ "/init" ]
