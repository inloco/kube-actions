ARG UBUNTU_VERSION

FROM ubuntu:${UBUNTU_VERSION}

ARG RUNNER_VERSION

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update -y && \
    apt-get install -y apt-transport-https busybox ca-certificates curl git gnupg-agent iputils-ping jq make software-properties-common sudo zstd && \
    busybox --install && \
    curl -L https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    apt-get update -y && \
    apt-get install -y docker-ce-cli && \
    COMPOSE_LATEST=$(curl -Ls https://api.github.com/repos/docker/compose/releases/latest | jq -r .tag_name) curl -Lo /usr/local/bin/docker-compose "https://github.com/docker/compose/releases/download/${COMPOSE_LATEST}/docker-compose-Linux-x86_64" && \
    chmod 0755 /usr/local/bin/docker-compose && \
    KUBERNETES_STABLE=$(curl -Ls https://storage.googleapis.com/kubernetes-release/release/stable.txt) curl -Lo /usr/local/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_STABLE}/bin/linux/amd64/kubectl" && \
    chmod 0755 /usr/local/bin/kubectl && \
    curl -Lo /tmp/awscli-exe-linux-x86_64.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip /tmp/awscli-exe-linux-x86_64.zip -d /tmp && \
    /tmp/aws/install && \
    useradd -u 1000 -m user && \
    echo 'user ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/user && \
    chmod 0440 /etc/sudoers.d/user && \
    mkdir /opt/actions-runner && \
    chown user:user /opt/actions-runner && \
    curl -L "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" | runuser -u user -- tar -vxzC /opt/actions-runner && \
    /opt/actions-runner/bin/installdependencies.sh && \
    rm -fR /var/lib/apt/lists/* /tmp/*

COPY ./init /init

ENTRYPOINT ["/init"]
