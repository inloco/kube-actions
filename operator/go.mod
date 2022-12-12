module github.com/inloco/kube-actions/operator

go 1.16

require (
	github.com/go-jose/go-jose/v3 v3.0.0
	github.com/go-logr/logr v1.2.3
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.3.0
	github.com/itchyny/gojq v0.12.6
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b3
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.11.0
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	golang.org/x/text v0.5.0
	k8s.io/api v0.26.0
	k8s.io/apimachinery v0.26.0
	k8s.io/client-go v0.26.0
	k8s.io/utils v0.0.0-20221107191617-1a15be271d1d
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	github.com/google/go-github/v32 => github.com/inloco/go-github/v32 v32.0.0-20200716220920-8f1b474407bc
	github.com/microsoft/azure-devops-go-api/azuredevops => github.com/inloco/azure-devops-go-api/azuredevops v0.0.0-20220303184925-16732b5bca92
	go.mozilla.org/pkcs7 => github.com/inloco/pkcs7 v0.0.0-20220127055019-e6b9106331c4
)
