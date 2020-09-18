FROM golang:1.15 AS build
WORKDIR /go/src/github.com/inloco/kube-actions/operator
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go mod download
COPY ./main.go ./main.go
COPY ./api/ ./api/
COPY ./constants/ ./constants/
COPY ./controllers/ ./controllers/
ARG VER
RUN CGO_ENABLED=0 go install -a -ldflags '-d -extldflags "-fno-PIC -static" -s -w -X github.com/inloco/kube-actions/operator/constants.ver='${VER} -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM gcr.io/distroless/static:nonroot AS runtime
COPY --from=build /go/bin/operator /manager
ENTRYPOINT ["/manager"]
