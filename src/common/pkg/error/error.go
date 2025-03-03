// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package error

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// https://github.com/grpc/grpc-go/blob/b3393d95a74e059d5663c758ec002df156a4091f/rpc_util.go#L994
var errContextCanceled = status.Error(codes.Canceled, context.Canceled.Error())

// IsContextCanceled is a helper that can detect both context.Canceled and the grpc-go context cancellation error, which
// is currently internal to google.golang.org/grpc/status. There is no explicit mapping between these two, since the
// server-sent context cancellation RPC error does not imply that the context used by the client would be cancelled.
// Thus, this check should only be used when distinguishing these cases is not important. For details, see
// https://github.com/grpc/grpc-go/issues/6862.
func IsContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, errContextCanceled)
}
