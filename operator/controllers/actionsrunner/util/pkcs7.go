package util

import (
	"crypto/x509/pkix"
	"encoding/asn1"

	"go.mozilla.org/pkcs7"
)

type ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type EncryptedData struct {
	Version              int
	EncryptedContentInfo EncryptedContentInfo
}

type EncryptedContentInfo struct {
	ContentType                asn1.ObjectIdentifier
	ContentEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedContent           []byte `asn1:"optional,tag:0"`
}

func DecryptPKCS7(data []byte, iv []byte, key []byte) ([]byte, error) {
	ed, err := asn1.Marshal(EncryptedData{
		Version: 0,
		EncryptedContentInfo: EncryptedContentInfo{
			ContentType: nil,
			ContentEncryptionAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm: pkcs7.OIDEncryptionAlgorithmAES128CBC,
				Parameters: asn1.RawValue{
					Class:      asn1.ClassContextSpecific,
					Tag:        0,
					IsCompound: true,
					Bytes:      iv,
				},
			},
			EncryptedContent: data,
		},
	})
	if err != nil {
		return nil, err
	}

	ci, err := asn1.Marshal(ContentInfo{
		ContentType: pkcs7.OIDEncryptedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      ed,
		},
	})
	if err != nil {
		return nil, err
	}

	cert, err := pkcs7.Parse(ci)
	if err != nil {
		return nil, err
	}

	return cert.DecryptUsingPSK(key)
}
