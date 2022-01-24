package util

import (
	"errors"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/microsoft/azure-devops-go-api/azuredevops/serviceendpoint"
)

type WellKnownServiceEndpointName string

const (
	WellKnownServiceEndpointNameSystemVssConnection WellKnownServiceEndpointName = "SystemVssConnection"
)

func GetSystemVssConnectionEndpoint(serviceEndpoints []serviceendpoint.ServiceEndpoint) (*serviceendpoint.ServiceEndpoint, error) {
	var serviceEndpoint *serviceendpoint.ServiceEndpoint
	for _, se := range serviceEndpoints {
		if name := se.Name; name == nil || *name != string(WellKnownServiceEndpointNameSystemVssConnection) {
			continue
		}

		serviceEndpoint = &se
		break
	}
	if serviceEndpoint == nil {
		return nil, errors.New("serviceEndpoint == nil")
	}

	return serviceEndpoint, nil
}

func GetServiceEndpointURL(serviceEndpoint *serviceendpoint.ServiceEndpoint) (string, error) {
	if serviceEndpoint == nil {
		return "", errors.New("serviceEndpoint == nil")
	}

	if serviceEndpoint.Url == nil {
		return "", errors.New("url == nil")
	}

	return *serviceEndpoint.Url, nil
}

func GetServiceEndpointAccessToken(serviceEndpoint *serviceendpoint.ServiceEndpoint) (string, error) {
	if serviceEndpoint == nil {
		return "", errors.New("serviceEndpoint == nil")
	}

	if serviceEndpoint.Authorization == nil {
		return "", errors.New("authorization == nil")
	}
	authorization := *serviceEndpoint.Authorization

	if authorization.Scheme == nil {
		return "", errors.New("scheme == nil")
	}
	scheme := *authorization.Scheme

	if scheme != "OAuth" {
		return "", errors.New(`scheme != "OAuth"`)
	}

	if authorization.Parameters == nil {
		return "", errors.New("parameters == nil")
	}
	parameters := *authorization.Parameters

	accessToken, ok := parameters["AccessToken"]
	if !ok {
		return "", errors.New(`parameters["AccessToken"] == nil`)
	}

	return accessToken, nil
}

func GetOrchestrationId(serviceEndpoints []serviceendpoint.ServiceEndpoint) (string, error) {
	serviceEndpoint, err := GetSystemVssConnectionEndpoint(serviceEndpoints)
	if err != nil {
		return "", err
	}

	accessTokenStr, err := GetServiceEndpointAccessToken(serviceEndpoint)
	if err != nil {
		return "", err
	}

	accessTokenJWT, err := jwt.ParseSigned(accessTokenStr)
	if err != nil {
		return "", err
	}

	var claims map[string]interface{}
	if err := accessTokenJWT.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return "", err
	}

	orchid, ok := claims["orchid"]
	if !ok {
		return "", errors.New(`claims["orchid"] == nil`)
	}

	orchestrationId, ok := orchid.(string)
	if !ok {
		return "", errors.New(`orchid.(string) == nil`)
	}

	return orchestrationId, nil
}
