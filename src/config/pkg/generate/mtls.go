// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generate

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/config/pkg/config"
)

type certificateData struct {
	certificate x509.Certificate
	privateKey  ed25519.PrivateKey
	publicKey   ed25519.PublicKey
}

type signedCaData struct {
	certificatePem string
	certificate    x509.Certificate
	privateKey     ed25519.PrivateKey
}

func (c *certificateData) selfSign() (*signedCaData, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, &c.certificate, &c.certificate, c.publicKey, c.privateKey)
	if err != nil {
		return nil, err
	}

	certPem, err := encodeCert(certBytes)
	if err != nil {
		return nil, err
	}

	return &signedCaData{
		certificatePem: certPem,
		certificate:    c.certificate,
		privateKey:     c.privateKey,
	}, nil
}

func (c *certificateData) toMTls(ca *signedCaData) (*config.MTlsConfig, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, &c.certificate, &ca.certificate, c.publicKey, ca.privateKey)
	if err != nil {
		return nil, err
	}

	certPem, err := encodeCert(certBytes)
	if err != nil {
		return nil, err
	}

	privKeyPem, err := encodePrivKey(c.privateKey)
	if err != nil {
		return nil, err
	}

	return &config.MTlsConfig{
		Ca:   ca.certificatePem,
		Key:  privKeyPem,
		Cert: certPem,
	}, nil
}

type certType int

const (
	ca certType = iota
	client
	server
)

func initCert(cType certType, validFor time.Duration) (*certificateData, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	var commonName string
	switch cType {
	case ca:
		commonName = "Supernetes mTLS root CA"
	case client:
		commonName = "Supernetes mTLS client (agent)"
	case server:
		commonName = "Supernetes mTLS server (controller)"
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	var keyUsage x509.KeyUsage
	switch cType {
	case ca:
		keyUsage |= x509.KeyUsageCertSign // CA is only used to sign client certs
	default:
		keyUsage |= x509.KeyUsageDigitalSignature // mTLS client/server certs
	}

	extKeyUsage := make([]x509.ExtKeyUsage, 0)
	switch cType {
	case ca:
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth)
	case client:
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageClientAuth)
	case server:
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageServerAuth)
	}

	var dnsNames []string
	switch cType {
	case server:
		dnsNames = append(dnsNames, supernetes.CertSANSupernetes)
	default:
	}

	certificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		DNSNames:              dnsNames,
		BasicConstraintsValid: cType == ca,
		IsCA:                  cType == ca,
	}

	return &certificateData{
		certificate: certificate,
		privateKey:  privKey,
		publicKey:   pubKey,
	}, nil
}

// MTls generates an mTLS configuration pair for a controller and an agent
func MTls(validFor time.Duration) (*config.MTlsConfig, *config.MTlsConfig, error) {
	caCert, err := initCert(ca, validFor)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize CA certificate: %w", err)
	}

	caSigned, err := caCert.selfSign()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to self-sign CA certificate: %w", err)
	}

	controllerCert, err := initCert(server, validFor)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize controller certificate: %w", err)
	}

	agentCert, err := initCert(client, validFor)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize agent certificate: %w", err)
	}

	controllerMTls, err := controllerCert.toMTls(caSigned)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create controller mTLS configuration: %w", err)
	}

	agentMTls, err := agentCert.toMTls(caSigned)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create agent mTLS configuration: %w", err)
	}

	return controllerMTls, agentMTls, nil
}
