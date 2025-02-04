// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package certificates

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"sort"

	"github.com/pkg/errors"
	certificates "k8s.io/api/certificates/v1"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/certificate"
	netutils "k8s.io/utils/net"
)

// NewKubeletServerCertificateManager creates a certificate
// manager for the kubelet for retrieving a server certificate.
func NewKubeletServerCertificateManager(kubeClient clientset.Interface, nodeName string, getAddresses func() []v1.NodeAddress, certDirectory string) (certificate.Manager, error) {
	certificateStore, err := certificate.NewFileStore(nodeName, certDirectory, certDirectory, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize server certificate store")
	}

	m, err := certificate.NewManager(&certificate.Config{
		ClientsetFn: func(_ *tls.Certificate) (clientset.Interface, error) {
			return kubeClient, nil
		},
		GetTemplate:      newGetTemplateFn(nodeName, getAddresses),
		SignerName:       certificates.KubeletServingSignerName,
		GetUsages:        certificate.DefaultKubeletServingGetUsages,
		CertificateStore: certificateStore,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize server certificate manager")
	}

	return m, nil
}

func newGetTemplateFn(nodeName string, getAddresses func() []v1.NodeAddress) func() *x509.CertificateRequest {
	return func() *x509.CertificateRequest {
		hostnames, ips := addressesToHostnamesAndIPs(getAddresses())
		// By default, require at least one IP before requesting a serving certificate
		hasRequiredAddresses := len(ips) > 0

		// Optionally, allow requesting a serving certificate with just a DNS name
		/*if utilfeature.DefaultFeatureGate.Enabled(features.AllowDNSOnlyNodeCSR) {
			hasRequiredAddresses = hasRequiredAddresses || len(hostnames) > 0
		}*/

		// Don't return a template if we have no addresses to request for
		if !hasRequiredAddresses {
			return nil
		}

		return &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   fmt.Sprintf("system:node:%s", nodeName),
				Organization: []string{"system:nodes"},
			},
			DNSNames:    hostnames,
			IPAddresses: ips,
		}
	}
}

func addressesToHostnamesAndIPs(addresses []v1.NodeAddress) (dnsNames []string, ips []net.IP) {
	seenDNSNames := make(map[string]bool)
	seenIPs := make(map[string]bool)

	for _, address := range addresses {
		if len(address.Address) == 0 {
			continue
		}

		switch address.Type {
		case v1.NodeHostName:
			if ip := netutils.ParseIPSloppy(address.Address); ip != nil {
				seenIPs[address.Address] = true
			} else {
				seenDNSNames[address.Address] = true
			}
		case v1.NodeExternalIP, v1.NodeInternalIP:
			if ip := netutils.ParseIPSloppy(address.Address); ip != nil {
				seenIPs[address.Address] = true
			}
		case v1.NodeExternalDNS, v1.NodeInternalDNS:
			seenDNSNames[address.Address] = true
		}
	}

	for dnsName := range seenDNSNames {
		dnsNames = append(dnsNames, dnsName)
	}
	for ip := range seenIPs {
		ips = append(ips, netutils.ParseIPSloppy(ip))
	}

	// Return in stable order
	sort.Strings(dnsNames)
	sort.Slice(ips, func(i, j int) bool { return ips[i].String() < ips[j].String() })

	return dnsNames, ips
}
