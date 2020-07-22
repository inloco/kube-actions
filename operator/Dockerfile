FROM golang:1.14-alpine AS build
WORKDIR /go/src/github.com/inloco/kube-actions/operator
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on
RUN apk add ca-certificates git && \
    go get -d -v ./... && \
    go install -a -installsuffix 'cgo' -gcflags 'all=-trimpath "/go/src"' -ldflags '-extldflags "-static" -s -w' -tags 'netgo' -v ./...

FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/bin/operator /init
ENTRYPOINT [ "/init" ]
