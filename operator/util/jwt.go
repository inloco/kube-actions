package util

import (
	"crypto/rsa"
	"time"

	"github.com/google/uuid"
	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
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
		return "", WrapErr(err)
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
		return "", WrapErr(err)
	}

	return token, nil
}
