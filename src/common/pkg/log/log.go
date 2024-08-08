// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Main logger instance
var logger *zerolog.Logger

func Init(level zerolog.Level) {
	if logger != nil {
		Panic().Msg("logger re-initialization is forbidden")
	}

	l := baseLogger(level).Caller().Logger()
	logger = &l
}

func baseLogger(level zerolog.Level) zerolog.Context {
	return zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime},
	).Level(level).With().Timestamp()
}

func getLogger() *zerolog.Logger {
	if logger == nil {
		Init(zerolog.PanicLevel)
		Panic().Msg("attempt to log with uninitialized logger")
	}

	return logger
}

func Trace() *zerolog.Event        { return getLogger().Trace() }
func Debug() *zerolog.Event        { return getLogger().Debug() }
func Info() *zerolog.Event         { return getLogger().Info() }
func Warn() *zerolog.Event         { return getLogger().Warn() }
func Error() *zerolog.Event        { return getLogger().Error() }
func Err(err error) *zerolog.Event { return getLogger().Err(err) }
func Fatal() *zerolog.Event        { return getLogger().Fatal() }
func Panic() *zerolog.Event        { return getLogger().Panic() }

// Scoped logger with custom fields
func Scoped() zerolog.Context { return getLogger().With() }
