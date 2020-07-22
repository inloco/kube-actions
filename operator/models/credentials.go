package models

type Credentials struct {
	Scheme string         `json:"scheme"`
	Data   CredentialData `json:"data"`
}

type CredentialData struct {
	ClientId         string `json:"clientId"`
	AuthorizationURL string `json:"authorizationUrl"`
	OAuthEndpointURL string `json:"oauthEndpointUrl"`
}
