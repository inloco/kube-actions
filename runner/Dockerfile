ARG UBUNTU_VERSION

FROM ubuntu:${UBUNTU_VERSION}

ARG RUNNER_VERSION

RUN export DEBIAN_FRONTEND=noninteractive && \
    export IFS="$(printf '\n\b')" && \
    export $(cat /etc/os-release | sed 's/"//g') && \
    unset IFS && \
    apt-get update && \
    apt-get install -y apt-transport-https busybox ca-certificates curl gettext-base git gnupg-agent iputils-ping jq make python3 python3-distutils software-properties-common sudo zstd && \
    curl -Lf https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    curl -Lf "https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/x${NAME}_${VERSION_ID}/Release.key" | sudo apt-key add - && \
    echo "deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/x${NAME}_${VERSION_ID}/ /" > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list && \
    apt-get update && \
    apt-get install -y docker-ce-cli skopeo && \
    busybox --install && \
    curl -Lf https://bootstrap.pypa.io/get-pip.py | python3 - && \
    export HUB_URL=$(curl -Lf https://api.github.com/repos/github/hub/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64")) | .browser_download_url') && \
    curl -Lf ${HUB_URL} | tar -xzvC /tmp && \
    /tmp/hub-*/install && \
    mkdir -p /usr/local/lib/docker/cli-plugins && \
    export APP_URL=$(curl -Lf https://api.github.com/repos/docker/app/releases | jq -r '.[0].assets[] | select(.name | match("linux.tar.gz$")) | .browser_download_url') && \
    curl -Lf ${APP_URL} | tar -xzvC /usr/local/lib/docker/cli-plugins && \
    mv /usr/local/lib/docker/cli-plugins/docker-app* /usr/local/lib/docker/cli-plugins/docker-app && \
    export BUILDX_URL=$(curl -Lf https://api.github.com/repos/docker/buildx/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64$")) | .browser_download_url') && \
    curl -Lfo /usr/local/lib/docker/cli-plugins/docker-buildx ${BUILDX_URL} && \
    chmod 0755 /usr/local/lib/docker/cli-plugins/docker-buildx && \
    export COMPOSE_URL=$(curl -Lf https://api.github.com/repos/docker/compose/releases/latest | jq -r '.assets[] | select(.name | match("Linux-x86_64$")) | .browser_download_url') && \
    curl -Lfo /usr/local/bin/docker-compose ${COMPOSE_URL} && \
    chmod 0755 /usr/local/bin/docker-compose && \
    export KUBERNETES_VERSION=$(curl -Lf https://storage.googleapis.com/kubernetes-release/release/stable.txt) && \
    curl -Lfo /usr/local/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl" && \
    chmod 0755 /usr/local/bin/kubectl && \
    curl -Lfo /usr/local/bin/ec2-metadata http://s3.amazonaws.com/ec2metadata/ec2-metadata && \
    chmod 0755 /usr/local/bin/ec2-metadata && \
    curl -Lfo /tmp/awscli-exe-linux-x86_64.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip /tmp/awscli-exe-linux-x86_64.zip -d /tmp && \
    /tmp/aws/install && \
    export ECRLOGIN_VERSION=$(curl -Lf https://api.github.com/repos/awslabs/amazon-ecr-credential-helper/releases/latest | jq -r '.tag_name' | sed 's/^v//g') && \
    curl -Lfo /usr/local/bin/docker-credential-ecr-login "https://amazon-ecr-credential-helper-releases.s3.us-east-2.amazonaws.com/${ECRLOGIN_VERSION}/linux-amd64/docker-credential-ecr-login" && \
    chmod 0755 /usr/local/bin/docker-credential-ecr-login && \
    pip3 install git-remote-codecommit && \
    useradd -u 1000 -m user && \
    echo 'user ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/user && \
    chmod 0440 /etc/sudoers.d/user && \
    mkdir /opt/actions-runner && \
    chown user:user /opt/actions-runner && \
    export RUNNER_URL=$(curl -Lf https://api.github.com/repos/actions/runner/releases | jq -r '.[] | select(.tag_name == "v'${RUNNER_VERSION}'") | .assets[] | select(.name | contains("linux-x64")) | .browser_download_url') && \
    curl -Lf ${RUNNER_URL} | runuser -u user -- tar -vxzC /opt/actions-runner && \
    /opt/actions-runner/bin/installdependencies.sh && \
    rm -fRv ${XDG_CACHE_HOME:-${HOME}/.cache}/pip/* /var/lib/apt/lists/* /tmp/*

COPY ./init /sbin/init

ENV ACTIONS_ALLOW_UNSECURE_COMMANDS true
ENV AWS_EC2_METADATA_DISABLED true
ENV AWS_SDK_LOAD_CONFIG 1
ENV BUILDX_NO_DEFAULT_LOAD false
ENV DOCKER_BUILDKIT 0
ENV DOCKER_CLI_EXPERIMENTAL enabled
ENV RUNNER_ALLOW_RUNASROOT true

# languages tools-specific
ENV GRADLE_OPTS -Dorg.gradle.daemon=false

ENTRYPOINT ["/sbin/init"]
