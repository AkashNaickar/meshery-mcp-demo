// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package meshery

import (
	"errors"
	"fmt"
	"net/http"
)

// JSON-RPC error codes mapped from Meshery failures. These are negative to
// avoid colliding with the MCP protocol's reserved codes.
const (
	// ErrCodeUnreachable maps to "Meshery Daemon Unreachable".
	ErrCodeUnreachable = -32001
	// ErrCodeAuthFailure maps to "Auth Failure".
	ErrCodeAuthFailure = -32002
	// ErrCodeInvalidParams maps to "Invalid Parameter Schemas".
	ErrCodeInvalidParams = -32602
	// ErrCodeUnknown is the fallback for unmapped Meshery errors.
	ErrCodeUnknown = -32603
)

// APIError is a typed error returned by the Meshery client. It carries a
// JSON-RPC-compatible code so the MCP layer can surface the right error to the
// agent without re-deriving it from the HTTP status.
type APIError struct {
	// Code is the JSON-RPC error code.
	Code int
	// Status is the HTTP status that produced the error, or 0.
	Status int
	// Op is the operation that failed (e.g. "deploy").
	Op string
	// Msg is the human-readable detail.
	Msg string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("meshery %s: %s", e.Op, e.Msg)
}

// Unwrap supports errors.Is/As over wrapped sentinel errors.
func (e *APIError) Unwrap() error { return nil }

// Sentinels allow callers to test for categories with errors.Is.
var (
	ErrUnreachable = errors.New("meshery daemon unreachable")
	ErrAuth        = errors.New("meshery auth failure")
)

// mapStatusToCode converts an HTTP status into a JSON-RPC error code.
func mapStatusToCode(status int) int {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrCodeAuthFailure
	case status >= 400 && status < 500:
		return ErrCodeInvalidParams
	case status >= 500:
		return ErrCodeUnreachable
	default:
		return ErrCodeUnknown
	}
}
