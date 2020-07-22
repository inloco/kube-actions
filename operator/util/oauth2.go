package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	keyContentType = "Content-Type"
	valContentType = "application/x-www-form-urlencoded"

	keyGrantType = "grant_type"
	valGrantType = "client_credentials"

	keyClientAssertionType = "client_assertion_type"
	valClientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	keyClientAssertion = "client_assertion"

	errInvalidRequest       = "invalid_request"
	errInvalidClient        = "invalid_client"
	errInvalidGrant         = "invalid_grant"
	errUnauthorizedClient   = "unauthorized_client"
	errUnsupportedGrantType = "unsupported_grant_type"
	errInvalidScope         = "invalid_scope"
)

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

type successfulResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type response struct {
	successfulResponse `json:",inline"`
	errorResponse      `json:",inline"`
}

func newRequest(ctx context.Context, tokenEndpoint string, clientAssertion string) (*http.Request, error) {
	body := url.Values{}
	body.Set(keyGrantType, valGrantType)
	body.Set(keyClientAssertionType, valClientAssertionType)
	body.Set(keyClientAssertion, clientAssertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, WrapErr(err)
	}
	req.Header.Set(keyContentType, valContentType)

	return req, nil
}

func fetchResponse(req *http.Request) (*response, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, WrapErr(err)
	}
	defer res.Body.Close()

	var body response
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, WrapErr(err)
	}

	return &body, nil
}

func unwrapErr(res errorResponse) error {
	if err := res.Error; err != "" {
		if desc := res.ErrorDescription; desc != "" {
			return fmt.Errorf("%v: %v", err, desc)
		}

		return errors.New(err)
	}

	return nil
}

func AccessToken(ctx context.Context, tokenEndpoint string, clientAssertion string) (string, error) {
	req, err := newRequest(ctx, tokenEndpoint, clientAssertion)
	if err != nil {
		return "", err
	}

	res, err := fetchResponse(req)
	if err != nil {
		return "", err
	}

	if err := unwrapErr(res.errorResponse); err != nil {
		return "", WrapErr(err)
	}

	return res.AccessToken, nil
}
