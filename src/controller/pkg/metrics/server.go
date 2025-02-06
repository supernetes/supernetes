// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supernetes/supernetes/common/pkg/log"
)

type Server interface {
	Start()
}

type server struct {
	server *http.Server
}

var _ http.Handler = &server{}

func NewServer(port int) Server {
	s := &server{}
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     nil,
	}

	return s
}

func (s *server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	b, err := io.ReadAll(request.Body)
	_ = request.Body.Close()
	log.Info().
		Err(err).
		Interface("URL", request.URL).
		Str("URI", request.RequestURI).
		Str("remote", request.RemoteAddr).
		Str("method", request.Method).
		Str("proto", request.Proto).
		Interface("headers", request.Header).
		Bytes("body", b).
		Msg("received request")
}

func (s *server) Start() {
	go func() {
		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Err(err).Msg("metrics server closed unexpectedly")
		}
	}()
}
