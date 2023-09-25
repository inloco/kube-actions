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

func (p *RSAParameters) ToRSAPrivateKey() (*rsa.PrivateKey, error) {
	if p.D == nil {
		return nil, errors.New(".D == nil")
	}

	if p.DP == nil {
		return nil, errors.New(".DP == nil")
	}

	if p.DQ == nil {
		return nil, errors.New(".DQ == nil")
	}

	if p.Exponent == nil {
		return nil, errors.New(".Exponent == nil")
	}

	if p.InverseQ == nil {
		return nil, errors.New(".InverseQ == nil")
	}

	if p.Modulus == nil {
		return nil, errors.New(".Modulus == nil")
	}

	if p.P == nil {
		return nil, errors.New(".P == nil")
	}

	if p.Q == nil {
		return nil, errors.New(".Q == nil")
	}

	privateKey := rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: new(big.Int).SetBytes(p.Modulus),
			E: int(new(big.Int).SetBytes(p.Exponent).Int64()),
		},
		D: new(big.Int).SetBytes(p.D),
		Primes: []*big.Int{
			new(big.Int).SetBytes(p.P),
			new(big.Int).SetBytes(p.Q),
		},
		Precomputed: rsa.PrecomputedValues{
			Dp:   new(big.Int).SetBytes(p.DP),
			Dq:   new(big.Int).SetBytes(p.DQ),
			Qinv: new(big.Int).SetBytes(p.InverseQ),
		},
	}

	return &privateKey, nil
}
