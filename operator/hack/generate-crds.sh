#!/bin/sh
set -ex

TEMP=$(mktemp -d)
trap "rm -fR ${TEMP}" EXIT SIGINT SIGTERM

ROOT=$(dirname "${BASH_SOURCE}")/..

LATEST=$(curl -Ls https://api.github.com/repos/operator-framework/operator-sdk/releases/latest | jq -r .tag_name)

case $(uname -sm) in
'Darwin x86_64')
    wget -O ${TEMP}/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${LATEST}/operator-sdk-${LATEST}-x86_64-apple-darwin
    ;;

'Linux x86_64')
    wget -O ${TEMP}/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${LATEST}/operator-sdk-${LATEST}-x86_64-linux-gnu
    ;;

*)
    echo "unknown kernel and/or architecture" >&2
    exit 1
    ;;
esac

chmod +x ${TEMP}/operator-sdk

cd $ROOT

mkdir -p ./build
touch ./build/Dockerfile

${TEMP}/operator-sdk generate crds
cp ./deploy/crds/inloco.com.br_actionsrunners_crd.yaml ./k8s/customResourceDefinition.yaml
