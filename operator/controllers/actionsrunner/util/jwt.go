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

package util

import (
	"crypto/rsa"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/uuid"
)

const (
	typJWT     = "JWT"
	valTimeMin = 5
	clkSkewMin = 1
)

func ClientAssertion(clientId string, tokenEndpoint string, key *rsa.PrivateKey) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       key,
		},
		new(jose.SignerOptions).WithType(typJWT),
	)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:    clientId,
		Subject:   clientId,
		Audience:  jwt.Audience{tokenEndpoint},
		Expiry:    jwt.NewNumericDate(now.Add((valTimeMin - clkSkewMin) * time.Minute)),
		NotBefore: jwt.NewNumericDate(now.Add(-clkSkewMin * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}

	token, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		return "", err
	}

	return token, nil
}
