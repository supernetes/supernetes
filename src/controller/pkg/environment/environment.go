// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package environment

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
)

// Environment exposes the dynamic (environment) configuration of the controller
type Environment interface {
	ControllerNamespace() *string          // Return the controller namespace, nil if unknown
	ControllerServiceAccountName() *string // Return the controller service account name, nil if unknown
	ControllerAddress() *netip.Addr        // Return the IP address of the controller, nil if unknown
}

type environment struct {
	controllerNamespace          string
	controllerServiceAccountName string
	controllerAddress            netip.Addr
}

// Load acquires and parses the dynamic configuration from the environment
func Load() Environment {
	controllerNamespace, err := loadString(supernetes.ControllerNamespace)
	if err != nil {
		log.Warn().Err(err).Msg("controller namespace unavailable")
	}

	controllerServiceAccountName, err := loadString(supernetes.ControllerServiceAccountName)
	if err != nil {
		log.Warn().Err(err).Msg("controller service account name unavailable")
	}

	controllerAddress, err := loadControllerAddress()
	if err != nil {
		log.Warn().Err(err).Msg("controller address unavailable")
	}

	return &environment{
		controllerNamespace:          controllerNamespace,
		controllerServiceAccountName: controllerServiceAccountName,
		controllerAddress:            controllerAddress,
	}
}

func (e *environment) ControllerNamespace() *string {
	if len(e.controllerNamespace) > 0 {
		return &e.controllerNamespace
	}

	return nil
}

func (e *environment) ControllerServiceAccountName() *string {
	if len(e.controllerServiceAccountName) > 0 {
		return &e.controllerServiceAccountName
	}

	return nil
}

func (e *environment) ControllerAddress() *netip.Addr {
	if e.controllerAddress.IsValid() {
		return &e.controllerAddress
	}

	return nil
}

func loadString(name string) (string, error) {
	env := os.Getenv(name)
	if len(env) == 0 {
		return "", fmt.Errorf("%s unset", name)
	}

	return env, nil
}

func loadControllerAddress() (netip.Addr, error) {
	// Take in status.PodIP, don't try to guess it here
	env := os.Getenv(supernetes.ControllerAddress)
	if len(env) == 0 {
		return netip.Addr{}, fmt.Errorf("%s unset", supernetes.ControllerAddress)
	}

	return netip.ParseAddr(env)
}
