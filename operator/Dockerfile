FROM golang:1.18.2 as build

RUN CGO_ENABLED=0 go get -v github.com/go-delve/delve/cmd/dlv

WORKDIR /go/src/github.com/inloco/kube-actions/operator
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY metrics metrics/
COPY constants constants/

ARG VER
ARG API
RUN CGO_ENABLED=0 go install -a -ldflags '-d -extldflags "-fno-PIC -static" -X github.com/inloco/kube-actions/operator/constants.ver='${VER}' -X github.com/inloco/kube-actions/operator/constants.api='${API} -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM gcr.io/distroless/static:nonroot
COPY --from=build /go/bin/dlv /usr/local/bin/dlv
COPY --from=build /go/bin/operator /manager
ENTRYPOINT ["/manager"]
