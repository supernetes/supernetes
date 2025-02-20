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

	certificates "k8s.io/api/certificates/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/certificate"
	compbasemetrics "k8s.io/component-base/metrics"
	netutils "k8s.io/utils/net"
)

const metricsNamespace = "supernetes_controller"

// NewKubeletServerCertificateManager creates a certificate manager for the kubelet when retrieving a server certificate
// or returns an error.
func NewKubeletServerCertificateManager(kubeClient clientset.Interface, nodeName types.NodeName, getAddresses func() []v1.NodeAddress, certDirectory string) (certificate.Manager, error) {
	var clientsetFn certificate.ClientsetFunc
	if kubeClient != nil {
		clientsetFn = func(current *tls.Certificate) (clientset.Interface, error) {
			return kubeClient, nil
		}
	}
	certificateStore, err := certificate.NewFileStore(
		string(nodeName),
		certDirectory,
		certDirectory,
		"",
		"")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server certificate store: %v", err)
	}
	var certificateRenewFailure = compbasemetrics.NewCounter(
		&compbasemetrics.CounterOpts{
			Namespace:      metricsNamespace,
			Subsystem:      string(nodeName), // TODO: Sanitization
			Name:           "server_expiration_renew_errors",
			Help:           "Counter of certificate renewal errors.",
			StabilityLevel: compbasemetrics.ALPHA,
		},
	)
	// TODO: This can panic, replace with .Register()
	//legacyregistry.MustRegister(certificateRenewFailure)

	certificateRotationAge := compbasemetrics.NewHistogram(
		&compbasemetrics.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: string(nodeName), // TODO: Sanitization
			Name:      "certificate_manager_server_rotation_seconds",
			Help:      "Histogram of the number of seconds the previous certificate lived before being rotated.",
			Buckets: []float64{
				60,        // 1  minute
				3600,      // 1  hour
				14400,     // 4  hours
				86400,     // 1  day
				604800,    // 1  week
				2592000,   // 1  month
				7776000,   // 3  months
				15552000,  // 6  months
				31104000,  // 1  year
				124416000, // 4  years
			},
			StabilityLevel: compbasemetrics.ALPHA,
		},
	)
	// TODO: This can panic, replace with .Register()
	//legacyregistry.MustRegister(certificateRotationAge)

	getTemplate := newGetTemplateFn(nodeName, getAddresses)

	m, err := certificate.NewManager(&certificate.Config{
		ClientsetFn:             clientsetFn,
		GetTemplate:             getTemplate,
		SignerName:              certificates.KubeletServingSignerName,
		GetUsages:               certificate.DefaultKubeletServingGetUsages,
		CertificateStore:        certificateStore,
		CertificateRotation:     certificateRotationAge,
		CertificateRenewFailure: certificateRenewFailure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server certificate manager: %v", err)
	}
	// TODO: This is deprecated
	//legacyregistry.RawMustRegister(compbasemetrics.NewGaugeFunc(
	//	&compbasemetrics.GaugeOpts{
	//		Namespace: metricsNamespace,
	//		Subsystem: string(nodeName), // TODO: Sanitization
	//		Name:      "certificate_manager_server_ttl_seconds",
	//		Help: "Gauge of the shortest TTL (time-to-live) of " +
	//			"the Kubelet's serving certificate. The value is in seconds " +
	//			"until certificate expiry (negative if already expired). If " +
	//			"serving certificate is invalid or unused, the value will " +
	//			"be +INF.",
	//		StabilityLevel: compbasemetrics.ALPHA,
	//	},
	//	func() float64 {
	//		if c := m.Current(); c != nil && c.Leaf != nil {
	//			return math.Trunc(time.Until(c.Leaf.NotAfter).Seconds())
	//		}
	//		return math.Inf(1)
	//	},
	//))

	return m, nil
}

func newGetTemplateFn(nodeName types.NodeName, getAddresses func() []v1.NodeAddress) func() *x509.CertificateRequest {
	return func() *x509.CertificateRequest {
		hostnames, ips := addressesToHostnamesAndIPs(getAddresses())
		// by default, require at least one IP before requesting a serving certificate
		hasRequiredAddresses := len(ips) > 0

		// optionally allow requesting a serving certificate with just a DNS name
		/*if utilfeature.DefaultFeatureGate.Enabled(features.AllowDNSOnlyNodeCSR) {
			hasRequiredAddresses = hasRequiredAddresses || len(hostnames) > 0
		}*/

		// don't return a template if we have no addresses to request for
		if !hasRequiredAddresses {
			return nil
		}
		return &x509.CertificateRequest{
			Subject: pkix.Name{
				// TODO: This doesn't match the CSR username (system:serviceaccount:supernetes:supernetes) which breaks cert-approver
				CommonName:   fmt.Sprintf("system:node:%s", nodeName),
				Organization: []string{"system:nodes"},
			},
			DNSNames:    hostnames,
			IPAddresses: ips,
		}
	}
}

func addressesToHostnamesAndIPs(addresses []v1.NodeAddress) (dnsNames []string, ips []net.IP) {
	seenDNSNames := map[string]bool{}
	seenIPs := map[string]bool{}
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

	// return in stable order
	sort.Strings(dnsNames)
	sort.Slice(ips, func(i, j int) bool { return ips[i].String() < ips[j].String() })

	return dnsNames, ips
}
