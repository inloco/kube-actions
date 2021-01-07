/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dot

type Credentials struct {
	Scheme string          `json:"scheme"`
	Data   CredentialsData `json:"data"`
}

type CredentialsData struct {
	ClientId                string `json:"clientId"`
	AuthorizationURL        string `json:"authorizationUrl"`
	OAuthEndpointURL        string `json:"oauthEndpointUrl"`
	RequireFipsCryptography string `json:"requireFipsCryptography"`
}
