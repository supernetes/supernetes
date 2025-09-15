// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/controller/pkg/certificates"
	vkauth "github.com/supernetes/supernetes/controller/pkg/vk/auth"
	vkapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-base/cli/flag"
)

var errServingCertTimeout = errors.New("timeout waiting for Kubelet serving certificate")

// KubeletServer provides the Kubelet HTTP functionality for a Virtual Kubelet instance
type KubeletServer struct {
	kubeClient    kubernetes.Interface
	handler       vkapi.PodHandlerConfig
	vkAuth        vkauth.Auth
	disableAuth   bool
	nodeName      string
	nodeAddresses func() []v1.NodeAddress
	port          atomic.Int32
	ready         chan struct{}
}

func NewKubeletServer(kubeClient kubernetes.Interface, handler vkapi.PodHandlerConfig, vkAuth vkauth.Auth, disableAuth bool, nodeName string, nodeAddresses func() []v1.NodeAddress) *KubeletServer {
	return &KubeletServer{
		kubeClient:    kubeClient,
		handler:       handler,
		vkAuth:        vkAuth,
		disableAuth:   disableAuth,
		nodeName:      nodeName,
		nodeAddresses: nodeAddresses,
		ready:         make(chan struct{}),
	}
}

func (s *KubeletServer) Run(ctx context.Context, log *zerolog.Logger) error {
	// Create a listener for the server letting the OS pick a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}

	// Close the listener when stopped
	defer func() {
		_ = listener.Close()
	}()

	// Fetch the allocated port number
	port := listener.Addr().(*net.TCPAddr).Port

	// Start a Kubelet server certificate manager tailored to the VK instance
	mgr, err := certificates.NewKubeletServerCertificateManager(
		s.kubeClient, s.nodeName, s.nodeAddresses, supernetes.CertsDir,
	)
	if err != nil {
		return err
	}

	log.Trace().Msg("starting Kubelet serving certificate manager")
	mgr.Start() // Non-blocking
	defer mgr.Stop()

	// TODO: Sensible timeout for slow-signing controller managers
	mgrCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Minute, errServingCertTimeout)
	defer cancel()

	log.Trace().Msg("waiting for Kubelet serving certificate")
	err = wait.PollUntilContextCancel(mgrCtx, time.Second, true, func(ctx context.Context) (done bool, err error) {
		return mgr.Current() != nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve Kubelet serving certificate: %w", err)
	}
	log.Trace().Msg("received Kubelet serving certificate")

	apiHandler := http.NewServeMux()
	apiHandler.Handle("/", vkapi.PodHandler(s.handler, false))

	vkAuth := nodeutil.NoAuth()
	if s.disableAuth {
		// The OpenShift/OKD dashboard and `oc` CLI do not pass any credentials when accessing the Kubelet API, so allow
		// anonymous access to all resources for now. There might be potential for a more fine-grained authentication
		// configuration, such as disabling auth just for container log retrieval, although that would require
		// duplicating and modifying the VK PodHandler logic here as well as indexing and covering all relevant routes.
		log.Debug().Msgf("warning: Kubelet HTTP server authentication disabled (OpenShift/OKD mode)")
	} else {
		vkAuth, err = s.vkAuth.VkAuth(s.nodeName)
		if err != nil {
			return err
		}
	}

	srv := &http.Server{
		IdleTimeout:  90 * time.Second,     // From Kubelet, matches http.DefaultTransport keep-alive timeout
		ReadTimeout:  4 * 60 * time.Minute, // From Kubelet
		WriteTimeout: 4 * 60 * time.Minute, // From Kubelet
		Handler:      nodeutil.WithAuth(vkAuth, apiHandler),
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequestClientCert,    // mTLS, also enabled by Kubelet
			MinVersion: flag.DefaultTLSVersion(), // As per Kubelet default
			GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return mgr.Current(), nil
			},
			// NOTE: This doesn't support rotation, but the CA is seemingly valid for 10 years by default
			ClientCAs: s.vkAuth.ClientCAPool(),
		},
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	s.port.Store(int32(port)) // Update port number
	close(s.ready)            // Mark readiness

	// Start the server
	if err := srv.ServeTLS(listener, "", ""); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *KubeletServer) Port() int32 {
	return s.port.Load()
}

func (s *KubeletServer) Ready() <-chan struct{} {
	return s.ready
}
