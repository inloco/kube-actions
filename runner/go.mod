module github.com/inloco/kube-actions/runner

go 1.16

require (
	github.com/aws/aws-sdk-go-v2/config v1.8.2
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.5.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.14
	github.com/containerd/containerd v1.5.6 // indirect
	github.com/docker/cli v20.10.8+incompatible
	github.com/docker/docker v20.10.8+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	google.golang.org/grpc v1.41.0 // indirect
)
