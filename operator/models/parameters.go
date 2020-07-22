package models

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"math/big"
)

type RSAParameters struct {
	D        []byte `json:"d"`
	DP       []byte `json:"dp"`
	DQ       []byte `json:"dq"`
	Exponent []byte `json:"exponent"`
	InverseQ []byte `json:"inverseQ"`
	Modulus  []byte `json:"modulus"`
	P        []byte `json:"p"`
	Q        []byte `json:"q"`
}

func (rsap *RSAParameters) ToRSAPrivateKey() (*rsa.PrivateKey, error) {
	if rsap.D == nil {
		return nil, errors.New(".D == nil")
	}

	if rsap.DP == nil {
		return nil, errors.New(".DP == nil")
	}

	if rsap.DQ == nil {
		return nil, errors.New(".DQ == nil")
	}

	if rsap.Exponent == nil {
		return nil, errors.New(".Exponent == nil")
	}

	if rsap.InverseQ == nil {
		return nil, errors.New(".InverseQ == nil")
	}

	if rsap.Modulus == nil {
		return nil, errors.New(".Modulus == nil")
	}

	if rsap.P == nil {
		return nil, errors.New(".P == nil")
	}

	if rsap.Q == nil {
		return nil, errors.New(".Q == nil")
	}

	privateKey := rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: new(big.Int).SetBytes(rsap.Modulus),
			E: int(new(big.Int).SetBytes(rsap.Exponent).Int64()),
		},
		D: new(big.Int).SetBytes(rsap.D),
		Primes: []*big.Int{
			new(big.Int).SetBytes(rsap.P),
			new(big.Int).SetBytes(rsap.Q),
		},
		Precomputed: rsa.PrecomputedValues{
			Dp:   new(big.Int).SetBytes(rsap.DP),
			Dq:   new(big.Int).SetBytes(rsap.DQ),
			Qinv: new(big.Int).SetBytes(rsap.InverseQ),
		},
	}

	return &privateKey, nil
}

func NewRSAParameters() (*RSAParameters, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	rsaParameters := RSAParameters{
		D:        key.D.Bytes(),
		DP:       key.Precomputed.Dp.Bytes(),
		DQ:       key.Precomputed.Dq.Bytes(),
		Exponent: big.NewInt(int64(key.E)).Bytes(),
		InverseQ: key.Precomputed.Qinv.Bytes(),
		Modulus:  key.N.Bytes(),
		P:        key.Primes[0].Bytes(),
		Q:        key.Primes[1].Bytes(),
	}

	return &rsaParameters, nil
}
