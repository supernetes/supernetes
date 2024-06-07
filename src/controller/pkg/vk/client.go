// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"errors"
	"fmt"
	"os"

	"github.com/supernetes/supernetes/common/pkg/log"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func newCoreV1Interface() (corev1.CoreV1Interface, error) {
	kubecfg, err := loadInClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load in-cluster configuration: %v", err)
	}

	if kubecfg == nil {
		kubecfg, err = loadKubeconfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load cluster configuration: %v", err)
		}
	}

	k8sClient, err := kubernetes.NewForConfig(kubecfg)
	if err != nil {
		return nil, fmt.Errorf("creating K8s client failed: %v", err)
	}

	return k8sClient.CoreV1(), nil
}

func loadInClusterConfig() (*rest.Config, error) {
	kubecfg, err := rest.InClusterConfig()
	if err != nil {
		if errors.Is(err, rest.ErrNotInCluster) {
			log.Warn().Err(err).Msg("")
			return nil, nil
		} else {
			return nil, err
		}
	}

	log.Debug().Msg("loaded in-cluster configuration")
	return kubecfg, nil
}

func loadKubeconfig() (*rest.Config, error) {
	kubecfgEnv := os.Getenv("KUBECONFIG")
	if len(kubecfgEnv) == 0 {
		return nil, errors.New("KUBECONFIG unset")
	}

	log.Debug().Msgf("loading cluster configuration from %q", kubecfgEnv)
	kubecfgFile, err := os.ReadFile(os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}

	clientCfg, err := clientcmd.NewClientConfigFromBytes(kubecfgFile)
	if err != nil {
		return nil, err
	}

	return clientCfg.ClientConfig()
}
