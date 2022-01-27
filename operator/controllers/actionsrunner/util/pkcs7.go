package util

import (
	"crypto/x509/pkix"
	"encoding/asn1"

	"go.mozilla.org/pkcs7"
)

const (
	TagZero = 0
)

func DecryptPKCS7(data []byte, iv []byte, key []byte) ([]byte, error) {
	eci := pkcs7.EncryptedContentInfo{
		ContentType: pkcs7.OIDData,
		ContentEncryptionAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm: pkcs7.OIDEncryptionAlgorithmAES128CBC,
			Parameters: asn1.RawValue{
				Class:      asn1.ClassContextSpecific,
				Tag:        TagZero,
				IsCompound: false,
				Bytes:      iv,
			},
		},
		EncryptedContent: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        TagZero,
			IsCompound: false,
			Bytes:      data,
		},
	}

	return eci.Decrypt(key)
}
