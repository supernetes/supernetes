// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import (
	"errors"
	"net/url"
)

// Hostname parses the hostname from an RFC-3986-compliant gRPC endpoint
func Hostname(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	hostname := u.Hostname()
	if hostname == "" {
		u, err = url.Parse("//" + endpoint)
		if err != nil {
			return "", err
		}

		hostname = u.Hostname()
	}

	if hostname == "" {
		return "", errors.New("unable to parse hostname")
	}

	return hostname, nil
}
