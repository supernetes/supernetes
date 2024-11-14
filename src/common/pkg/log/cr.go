// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
)

// CRLogger provides a controller-runtime-compatible logging interface
func CRLogger(scope *zerolog.Logger) logr.Logger {
	if scope == nil {
		scope = getLogger()
	}

	l := baseLogger(scope.GetLevel()).CallerWithSkipFrameCount(2).Logger()
	l.UpdateContext(func(_ zerolog.Context) zerolog.Context {
		return scope.With() // Clone initial context from given logger
	})

	return zerologr.New(&l)
}
