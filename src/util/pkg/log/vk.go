// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package log

import (
	"fmt"

	"github.com/rs/zerolog"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
)

type vkLogger struct {
	l            zerolog.Logger
	fields       vklog.Fields
	errors       []error
	clampToDebug bool
}

// VKLogger provides a Virtual Kubelet Logger-compatible logging interface. clampToDebug can be used to map Info
// messages to be Debug messages, assuming that the log level permits Info, to decrease Virtual Kubelet verbosity.
func VKLogger(scope *zerolog.Logger, clampToDebug bool) vklog.Logger {
	if scope == nil {
		scope = getLogger()
	}

	l := baseLogger(scope.GetLevel()).CallerWithSkipFrameCount(2).Logger()
	l.UpdateContext(func(_ zerolog.Context) zerolog.Context {
		return scope.With() // Clone initial context from given logger
	})

	return &vkLogger{
		l:            l.With().CallerWithSkipFrameCount(4).Str("scope", "virtual-kubelet").Logger(),
		clampToDebug: clampToDebug,
	}
}

func (v *vkLogger) Debug(i ...interface{}) {
	v.msg(v.l.Debug(), fmt.Sprint(i...))
}

func (v *vkLogger) Debugf(s string, i ...interface{}) {
	v.msg(v.l.Debug(), fmt.Sprintf(s, i...))
}

func (v *vkLogger) Info(i ...interface{}) {
	if v.clampToDebug {
		v.Debug(i...)
		return
	}

	v.msg(v.l.Info(), fmt.Sprint(i...))
}

func (v *vkLogger) Infof(s string, i ...interface{}) {
	if v.clampToDebug {
		v.Debugf(s, i...)
		return
	}

	v.msg(v.l.Info(), fmt.Sprintf(s, i...))
}

func (v *vkLogger) Warn(i ...interface{}) {
	v.msg(v.l.Warn(), fmt.Sprint(i...))
}

func (v *vkLogger) Warnf(s string, i ...interface{}) {
	v.msg(v.l.Warn(), fmt.Sprintf(s, i...))
}

func (v *vkLogger) Error(i ...interface{}) {
	v.msg(v.l.Error(), fmt.Sprint(i...))
}

func (v *vkLogger) Errorf(s string, i ...interface{}) {
	v.msg(v.l.Error(), fmt.Sprintf(s, i...))
}

func (v *vkLogger) Fatal(i ...interface{}) {
	v.msg(v.l.Fatal(), fmt.Sprint(i...))
}

func (v *vkLogger) Fatalf(s string, i ...interface{}) {
	v.msg(v.l.Fatal(), fmt.Sprintf(s, i...))
}

func (v *vkLogger) WithField(s string, i interface{}) vklog.Logger {
	l := v.copy()

	if l.fields == nil {
		l.fields = make(vklog.Fields)
	}

	l.fields[s] = i

	return l
}

func (v *vkLogger) WithFields(fields vklog.Fields) vklog.Logger {
	l := v.copy()

	if l.fields == nil {
		l.fields = make(vklog.Fields)
	}

	for k, f := range fields {
		l.fields[k] = f
	}

	return l
}

func (v *vkLogger) WithError(err error) vklog.Logger {
	l := v.copy()
	l.errors = append(l.errors, err)
	return l
}

func (v *vkLogger) msg(e *zerolog.Event, s string) {
	e = e.Fields(v.fields)

	switch len(v.errors) {
	case 0:
	case 1:
		e = e.Err(v.errors[0])
	default:
		e = e.Errs(fmt.Sprintf("%ss", zerolog.ErrorFieldName), v.errors)
	}

	e.Msg(s)
}

func (v *vkLogger) copy() *vkLogger {
	l := &vkLogger{l: v.l, clampToDebug: v.clampToDebug}

	if v.fields != nil {
		l.fields = make(vklog.Fields)
		for k, f := range v.fields {
			l.fields[k] = f
		}
	}

	if v.errors != nil {
		l.errors = make([]error, len(v.errors))
		copy(l.errors, v.errors)
	}

	return l
}
