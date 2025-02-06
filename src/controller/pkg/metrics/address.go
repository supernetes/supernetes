// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

import (
	"errors"
	"fmt"
	"net/netip"
	"os"

	"github.com/supernetes/supernetes/common/pkg/supernetes"
)

// Config gathers the configuration required for the API server to retrieve node metrics from the
// Virtual Kubelet nodes. It may be nil, which indicates that the metrics interface is unavailable.
type Config interface {
	ControllerAddress() netip.Addr
}

type config struct {
	addr netip.Addr
}

var Unavailable = errors.New("metrics unavailable")

func NewConfig() (Config, error) {
	addr, err := loadControllerAddress()
	if err != nil {
		// Disable metrics (return a specific error) if no address is available
		return nil, fmt.Errorf("%w: %w", Unavailable, err)
	}

	return &config{
		addr: addr,
	}, nil
}

func (c *config) ControllerAddress() netip.Addr {
	return c.addr
}

func loadControllerAddress() (netip.Addr, error) {
	// Take in status.PodIP, don't try to guess it here
	env := os.Getenv(supernetes.ControllerAddress)
	if len(env) == 0 {
		return netip.Addr{}, fmt.Errorf("%s unset", supernetes.ControllerAddress)
	}

	return netip.ParseAddr(env)
}
