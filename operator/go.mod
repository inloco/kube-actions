module github.com/inloco/kube-actions/operator

go 1.15

require (
	github.com/go-logr/logr v0.2.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.1.4
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b3
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.18.6
	k8s.io/utils v0.0.0-20200716102541-988ee3149bb2
	sigs.k8s.io/controller-runtime v0.6.0
)

replace github.com/google/go-github/v32 => github.com/inloco/go-github/v32 v32.0.0-20200716220920-8f1b474407bc

replace github.com/microsoft/azure-devops-go-api/azuredevops => github.com/inloco/azure-devops-go-api/azuredevops v0.0.0-20210107205147-d721430e92a7
