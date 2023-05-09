ARG UBUNTU_VERSION

FROM golang:1.20-alpine AS build
RUN CGO_ENABLED=0 go install -v github.com/go-delve/delve/cmd/dlv@latest
WORKDIR /go/src/github.com/inloco/kube-actions/runner
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./main.go ./util.go ./
RUN CGO_ENABLED=0 go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM ubuntu:${UBUNTU_VERSION}
ARG UBUNTU_VERSION_NUMBER
ARG RUNNER_VERSION

# Follow config for hosted runners
ENV ImageOS ubuntu${UBUNTU_VERSION_NUMBER}
ENV RUNNER_TOOL_CACHE /opt/hostedtoolcache

ENV ACTIONS_ALLOW_UNSECURE_COMMANDS true
ENV ACTIONS_RUNNER_PRINT_LOG_TO_STDOUT true
ENV AWS_EC2_METADATA_DISABLED true
ENV AWS_SDK_LOAD_CONFIG 1
ENV BUILDX_NO_DEFAULT_LOAD false
ENV DOCKER_BUILDKIT 0
ENV DOCKER_CLI_EXPERIMENTAL enabled
ENV RUNNER_ALLOW_RUNASROOT true

RUN set -x && \
    export DEBIAN_FRONTEND=noninteractive && \
    export IFS="$(printf '\n\b')" && \
    export $(cat /etc/os-release | sed 's/"//g') && \
    unset IFS && \
    apt-get update && \
    apt-get install -y apt-transport-https apt-utils busybox ca-certificates curl gettext-base git gnupg-agent iputils-ping jq make parallel python3 python3-distutils software-properties-common ssh sudo zstd && \
    CURL="curl -sSLf --retry 3" && \
    $CURL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    $CURL https://cli.github.com/packages/githubcli-archive-keyring.gpg | apt-key add - && \
    add-apt-repository 'deb [arch=amd64] https://cli.github.com/packages stable main' && \
    export DEBIAN_MIRROR=http://deb.debian.org/debian && \
    export NETAVARK_PACKAGE=netavark && \
    export NETAVARK_VERSION=1.4.0-3 && \
    $CURL -o "/tmp/${NETAVARK_PACKAGE}.deb" "${DEBIAN_MIRROR}/pool/main/n/${NETAVARK_PACKAGE}/${NETAVARK_PACKAGE}_${NETAVARK_VERSION}_amd64.deb" && \
    export GGCI_PACKAGE=golang-github-containers-image && \
    export GGCI_VERSION=5.10.3-1 && \
    $CURL -o "/tmp/${GGCI_PACKAGE}.deb" "${DEBIAN_MIRROR}/pool/main/g/${GGCI_PACKAGE}/${GGCI_PACKAGE}_${GGCI_VERSION}_all.deb" && \
    export GGCC_PACKAGE=golang-github-containers-common && \
    export GGCC_VERSION=0.51.0+ds1-1 && \
    $CURL -o "/tmp/${GGCC_PACKAGE}.deb" "${DEBIAN_MIRROR}/pool/main/g/${GGCC_PACKAGE}/${GGCC_PACKAGE}_${GGCC_VERSION}_all.deb" && \
    export SKOPEO_PACKAGE=skopeo && \
    export SKOPEO_VERSION=1.9.3+ds1-1+b4 && \
    $CURL -o "/tmp/${SKOPEO_PACKAGE}.deb" "${DEBIAN_MIRROR}/pool/main/s/${SKOPEO_PACKAGE}/${SKOPEO_PACKAGE}_${SKOPEO_VERSION}_amd64.deb" && \
    apt-get update && \
    apt-get install -y docker-ce-cli gh "/tmp/${NETAVARK_PACKAGE}.deb" "/tmp/${GGCI_PACKAGE}.deb" "/tmp/${GGCC_PACKAGE}.deb" "/tmp/${SKOPEO_PACKAGE}.deb" && \
    busybox --install && \
    $CURL https://bootstrap.pypa.io/get-pip.py | python3 - && \
    export HUB_URL=$($CURL https://api.github.com/repos/github/hub/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64")) | .browser_download_url') && \
    $CURL ${HUB_URL} | tar -xzvC /tmp && \
    /tmp/hub-*/install && \
    mkdir -p /usr/local/lib/docker/cli-plugins && \
    export APP_URL=$($CURL https://api.github.com/repos/docker/app/releases | jq -r '.[0].assets[] | select(.name | match("linux.tar.gz$")) | .browser_download_url') && \
    $CURL ${APP_URL} | tar -xzvC /usr/local/lib/docker/cli-plugins && \
    mv /usr/local/lib/docker/cli-plugins/docker-app* /usr/local/lib/docker/cli-plugins/docker-app && \
    export BUILDX_URL=$($CURL https://api.github.com/repos/docker/buildx/releases/latest | jq -r '.assets[] | select(.name | match("linux-amd64$")) | .browser_download_url') && \
    $CURL -o /usr/local/lib/docker/cli-plugins/docker-buildx ${BUILDX_URL} && \
    chmod 0755 /usr/local/lib/docker/cli-plugins/docker-buildx && \
    $CURL -o /usr/local/bin/docker-compose https://github.com/docker/compose/releases/download/1.29.2/docker-compose-Linux-x86_64 && \
    chmod 0755 /usr/local/bin/docker-compose && \
    export KUBERNETES_VERSION=$($CURL https://storage.googleapis.com/kubernetes-release/release/stable.txt) && \
    $CURL -o /usr/local/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl" && \
    chmod 0755 /usr/local/bin/kubectl && \
    $CURL https://github.com/kubernetes-sigs/kustomize/raw/master/hack/install_kustomize.sh | bash -s /usr/local/bin && \
    chmod 0755 /usr/local/bin/kustomize && \
    $CURL -o /usr/local/bin/sops https://github.com/mozilla/sops/releases/download/v3.7.1/sops-v3.7.1.linux && \
    chmod 0755 /usr/local/bin/sops && \
    $CURL -o /usr/local/bin/ec2-metadata http://s3.amazonaws.com/ec2metadata/ec2-metadata && \
    chmod 0755 /usr/local/bin/ec2-metadata && \
    $CURL -o /tmp/awscli2.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip /tmp/awscli2.zip -d /tmp && \
    /tmp/aws/install && \
    export ECRLOGIN_VERSION=$($CURL https://api.github.com/repos/awslabs/amazon-ecr-credential-helper/releases/latest | jq -r '.tag_name' | sed 's/^v//g') && \
    $CURL -o /usr/local/bin/docker-credential-ecr-login "https://amazon-ecr-credential-helper-releases.s3.us-east-2.amazonaws.com/${ECRLOGIN_VERSION}/linux-amd64/docker-credential-ecr-login" && \
    chmod 0755 /usr/local/bin/docker-credential-ecr-login && \
    pip3 install git-remote-codecommit yq && \
    export CODECOMMIT_ENDPOINTS="$($CURL https://ip-ranges.amazonaws.com/ip-ranges.json | jq -r '.prefixes | unique_by(.region) | map("git-codecommit.\(.region).amazonaws.com") | join(" ")')" && \
    ssh-keyscan github.com ${CODECOMMIT_ENDPOINTS} >> /etc/ssh/ssh_known_hosts && \
    useradd -u 1000 -m user && \
    echo 'user ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/user && \
    chmod 0440 /etc/sudoers.d/user && \
    mkdir /opt/actions-runner && \
    chown user:user /opt/actions-runner && \
    export RUNNER_URL=$($CURL https://api.github.com/repos/actions/runner/releases | jq -r '.[] | select(.tag_name == "v'${RUNNER_VERSION}'") | .assets[] | select(.name | endswith("linux-x64-'${RUNNER_VERSION}'.tar.gz")) | .browser_download_url') && \
    $CURL ${RUNNER_URL} | runuser -u user -- tar -vxzC /opt/actions-runner && \
    /opt/actions-runner/bin/installdependencies.sh && \
    # Support Node 12 https://github.com/nodejs/node/discussions/43184
    sed -i "s/^providers = provider_sect$/# providers = provider_sect/" /etc/ssl/openssl.cnf && \
    rm -fRv ${XDG_CACHE_HOME:-${HOME}/.cache}/pip/* /var/lib/apt/lists/* /tmp/* && \
    mkdir -p ${RUNNER_TOOL_CACHE} && \
    chown user:user ${RUNNER_TOOL_CACHE}

COPY --from=build /go/bin/dlv /usr/local/bin/dlv
COPY --from=build /go/bin/runner /sbin/init
ENTRYPOINT ["/sbin/init"]
