// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package run

import (
	"os"
	"time"

	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/config/pkg/config"
	"github.com/supernetes/supernetes/config/pkg/generate"
	"github.com/supernetes/supernetes/util/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenerateFlags struct {
	GenerateOptions
}

func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{}
}

type GenerateOptions struct {
	AgentConfigPath string
	AgentEndpoint   string

	ControllerConfigPath      string
	ControllerPort            uint16
	ControllerSecret          bool
	ControllerSecretName      string
	ControllerSecretNamespace string

	CertDaysValid uint32
}

func (gf *GenerateFlags) NewGenerateOptions(_ []string, _ *pflag.FlagSet) (*GenerateOptions, error) {
	return &gf.GenerateOptions, nil
}

func Generate(g *GenerateOptions) error {
	log.Debug().Msg("creating bonded mTLS configuration for controller and agent")
	validityPeriod := time.Duration(g.CertDaysValid) * time.Hour * 24
	controllerMTls, agentMTls, err := generate.MTls(validityPeriod)
	if err != nil {
		return err
	}

	controllerConfig := &config.ControllerConfig{
		Port:       g.ControllerPort,
		MTlsConfig: *controllerMTls,
	}

	log.Debug().Msg("encoding controller configuration")
	var controllerConfigBytes []byte
	if g.ControllerSecret {
		var secret *corev1.Secret
		secret, err = controllerConfig.ToSecret(metav1.ObjectMeta{
			Name:      g.ControllerSecretName,
			Namespace: g.ControllerSecretNamespace,
		})
		if err != nil {
			return err
		}

		controllerConfigBytes, err = config.EncodeK8s(secret)
	} else {
		controllerConfigBytes, err = config.Encode(controllerConfig)
	}
	if err != nil {
		return err
	}

	log.Debug().Msg("encoding agent configuration")
	agentConfig, err := config.Encode(&config.AgentConfig{
		Endpoint:   g.AgentEndpoint,
		MTlsConfig: *agentMTls,
	})
	if err != nil {
		return err
	}

	log.Info().
		Str("path", g.ControllerConfigPath).
		Bool("secret", g.ControllerSecret).
		Msg("writing controller configuration")
	if err := writeConfig(g.ControllerConfigPath, controllerConfigBytes); err != nil {
		return err
	}

	log.Info().Str("path", g.AgentConfigPath).Msg("writing agent configuration")
	if err := writeConfig(g.AgentConfigPath, agentConfig); err != nil {
		return err
	}

	return nil
}

func writeConfig(path string, data []byte) error {
	err := os.Chmod(path, 0600)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
