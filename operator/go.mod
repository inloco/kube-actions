module github.com/inloco/kube-actions/operator

go 1.16

require (
	github.com/go-jose/go-jose/v3 v3.0.0
	github.com/go-logr/logr v1.2.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.3.0
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b3
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.11.0
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v0.23.1
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	github.com/google/go-github/v32 => github.com/inloco/go-github/v32 v32.0.0-20200716220920-8f1b474407bc
	github.com/microsoft/azure-devops-go-api/azuredevops => github.com/inloco/azure-devops-go-api/azuredevops v0.0.0-20210107205147-d721430e92a7
)
