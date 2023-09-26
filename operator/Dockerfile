FROM golang:1.20-alpine as build
RUN CGO_ENABLED=0 go install -v github.com/go-delve/delve/cmd/dlv@latest
WORKDIR /workspace
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
ARG VER
ARG API
RUN CGO_ENABLED=0 go build -a -o manager -ldflags '-d -extldflags "-fno-PIC -static" -X github.com/inloco/kube-actions/operator/constants.ver='${VER}' -X github.com/inloco/kube-actions/operator/constants.api='${API} -tags 'netgo osusergo static_build' -trimpath -v cmd/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=build /go/bin/dlv /usr/local/bin/dlv
COPY --from=build /workspace/manager /sbin/init
ENTRYPOINT ["/sbin/init"]
