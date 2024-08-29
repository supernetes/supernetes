// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generate

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
)

func encodeBytes(blockType string, blockData []byte) (string, error) {
	certPem := new(bytes.Buffer)
	if err := pem.Encode(certPem, &pem.Block{
		Type:  blockType,
		Bytes: blockData,
	}); err != nil {
		return "", err
	}
	return certPem.String(), nil
}

func encodeCert(certBytes []byte) (string, error) {
	return encodeBytes("CERTIFICATE", certBytes)
}

func encodePrivKey(key any) (string, error) {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	return encodeBytes("PRIVATE KEY", keyBytes)
}
