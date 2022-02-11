ARG UBUNTU_VERSION

FROM golang:1.17.7-alpine AS build
WORKDIR /go/src/github.com/inloco/kube-actions/runner
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./main.go ./util.go ./
RUN CGO_ENABLED=0 go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM ubuntu:${UBUNTU_VERSION}
ARG RUNNER_VERSION

RUN set -x && \
    export DEBIAN_FRONTEND=noninteractive && \
    export IFS="$(printf '\n\b')" && \
    export $(cat /etc/os-release | sed 's/"//g') && \
    unset IFS && \
    apt-get update && \
    apt-get install -y apt-transport-https busybox ca-certificates curl gettext-base git gnupg-agent iputils-ping jq make python3 python3-distutils software-properties-common sudo zstd parallel && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-key C99B11DEB97541F0 && \
    apt-add-repository https://cli.github.com/packages && \
    curl -sSLf https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    apt-get update && \
    apt-get install -y docker-ce-cli gh && \
    busybox --install && \
    curl -sSLf https://bootstrap.pypa.io/get-pip.py | python3 - && \
    export HUB_URL=$(curl -sSLf https://api.github.com/repos/github/hub/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64")) | .browser_download_url') && \
    curl -sSLf ${HUB_URL} | tar -xzvC /tmp && \
    /tmp/hub-*/install && \
    mkdir -p /usr/local/lib/docker/cli-plugins && \
    export APP_URL=$(curl -sSLf https://api.github.com/repos/docker/app/releases | jq -r '.[0].assets[] | select(.name | match("linux.tar.gz$")) | .browser_download_url') && \
    curl -sSLf ${APP_URL} | tar -xzvC /usr/local/lib/docker/cli-plugins && \
    mv /usr/local/lib/docker/cli-plugins/docker-app* /usr/local/lib/docker/cli-plugins/docker-app && \
    export BUILDX_URL=$(curl -sSLf https://api.github.com/repos/docker/buildx/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64$")) | .browser_download_url') && \
    curl -sSLfo /usr/local/lib/docker/cli-plugins/docker-buildx ${BUILDX_URL} && \
    chmod 0755 /usr/local/lib/docker/cli-plugins/docker-buildx && \
    curl -sSLfo /usr/local/bin/docker-compose https://github.com/docker/compose/releases/download/1.29.2/docker-compose-Linux-x86_64 && \
    chmod 0755 /usr/local/bin/docker-compose && \
    export KUBERNETES_VERSION=$(curl -sSLf https://storage.googleapis.com/kubernetes-release/release/stable.txt) && \
    curl -sSLfo /usr/local/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl" && \
    chmod 0755 /usr/local/bin/kubectl && \
    curl -sSLf https://github.com/kubernetes-sigs/kustomize/raw/master/hack/install_kustomize.sh | bash -s /usr/local/bin && \
    chmod 0755 /usr/local/bin/kustomize && \
    curl -sSLfo /usr/local/bin/sops https://github.com/mozilla/sops/releases/download/v3.7.1/sops-v3.7.1.linux && \
    chmod 0755 /usr/local/bin/sops && \
    curl -sSLfo /usr/local/bin/ec2-metadata http://s3.amazonaws.com/ec2metadata/ec2-metadata && \
    chmod 0755 /usr/local/bin/ec2-metadata && \
    curl -sSLfo /tmp/awscli2.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip /tmp/awscli2.zip -d /tmp && \
    /tmp/aws/install && \
    export ECRLOGIN_VERSION=$(curl -sSLf https://api.github.com/repos/awslabs/amazon-ecr-credential-helper/releases/latest | jq -r '.tag_name' | sed 's/^v//g') && \
    curl -sSLfo /usr/local/bin/docker-credential-ecr-login "https://amazon-ecr-credential-helper-releases.s3.us-east-2.amazonaws.com/${ECRLOGIN_VERSION}/linux-amd64/docker-credential-ecr-login" && \
    chmod 0755 /usr/local/bin/docker-credential-ecr-login && \
    pip3 install git-remote-codecommit yq && \
    useradd -u 1000 -m user && \
    echo 'user ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/user && \
    chmod 0440 /etc/sudoers.d/user && \
    mkdir /opt/actions-runner && \
    chown user:user /opt/actions-runner && \
    export RUNNER_URL=$(curl -sSLf https://api.github.com/repos/actions/runner/releases | jq -r '.[] | select(.tag_name == "v'${RUNNER_VERSION}'") | .assets[] | select(.name | endswith("linux-x64-'${RUNNER_VERSION}'.tar.gz")) | .browser_download_url') && \
    curl -sSLf ${RUNNER_URL} | runuser -u user -- tar -vxzC /opt/actions-runner && \
    /opt/actions-runner/bin/installdependencies.sh && \
    rm -fRv ${XDG_CACHE_HOME:-${HOME}/.cache}/pip/* /var/lib/apt/lists/* /tmp/*

COPY --from=build /go/bin/runner /sbin/runner

ENV ACTIONS_ALLOW_UNSECURE_COMMANDS true
ENV AWS_EC2_METADATA_DISABLED true
ENV AWS_SDK_LOAD_CONFIG 1
ENV BUILDX_NO_DEFAULT_LOAD false
ENV DOCKER_BUILDKIT 0
ENV DOCKER_CLI_EXPERIMENTAL enabled
ENV RUNNER_ALLOW_RUNASROOT true

ENTRYPOINT ["/sbin/runner"]
