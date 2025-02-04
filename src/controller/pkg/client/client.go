// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package client

import (
	"os"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/common/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewKubeConfig() (*rest.Config, error) {
	kubecfg, err := loadInClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load in-cluster configuration")
	}

	if kubecfg == nil {
		kubecfg, err = loadKubeconfig()
		if err != nil {
			return nil, errors.Wrap(err, "unable to load cluster configuration")
		}
	}

	// TODO: These need to be configurable
	kubecfg.QPS = 100 * rest.DefaultQPS
	kubecfg.Burst = 100 * rest.DefaultBurst

	return kubecfg, nil
}

func NewKubeClient(config *rest.Config) (kubernetes.Interface, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate Kubernetes client: %v")
	}
	return client, nil
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
