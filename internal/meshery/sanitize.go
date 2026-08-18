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
	"encoding/json"
	"regexp"
	"strings"
)

// SecretRedaction is the replacement used for values that must not reach the
// AI agent. Using a fixed marker is deliberate: variable-length redaction
// strings would make diffing and fingerprinting unreliable.
const SecretRedaction = "[REDACTED]"

// sensitiveValuePatterns match common credential-shaped values so the
// redaction engine can scrub them even when they appear outside a labeled
// field (e.g. a bearer token in a "value" slot).
var sensitiveValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\beyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\b`), // JWT
	regexp.MustCompile(`(?i)\b(?:ssh-rsa|ssh-ed25519|ecdsa-sha2-nistp256|BEGIN [A-Z ]*PRIVATE KEY)\b`),
	regexp.MustCompile(`(?i)\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`), // AWS access key
	regexp.MustCompile(`(?i)\bghp_[A-Za-z0-9]{36}\b`),       // GitHub PAT
}

// Sanitizer redacts credentials and Kubernetes Secret data from structured
// payloads before they are returned to the MCP client.
type Sanitizer struct {
	// maxDepth bounds recursion to protect against deeply nested or cyclic
	// JSON payloads from the Meshery API.
	maxDepth int
}

// NewSanitizer returns a Sanitizer with safe defaults.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{maxDepth: 32}
}

// SanitizeAny redacts sensitive values in place across maps, slices, strings,
// and JSON-encoded text. It returns the input so callers can chain.
func (s *Sanitizer) SanitizeAny(v any) any {
	return s.sanitizeValue(v, 0)
}

// SanitizeJSON redacts sensitive values from an arbitrary JSON document.
// It returns the redacted document encoded back to JSON, or the original bytes
// when the input is not valid JSON.
func (s *Sanitizer) SanitizeJSON(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	v = s.sanitizeValue(v, 0)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

// SanitizeJSON redacts sensitive values from a JSON document. It is a
// convenience method on Client so resource handlers can sanitize payloads
// without constructing their own Sanitizer.
func (c *Client) SanitizeJSON(raw []byte) []byte {
	return NewSanitizer().SanitizeJSON(raw)
}

// Sanitize redacts sensitive values from a decoded value (map/slice/string).
func (c *Client) Sanitize(v any) any {
	return NewSanitizer().SanitizeAny(v)
}

// sanitizeValue recursively redacts a decoded JSON value.
func (s *Sanitizer) sanitizeValue(v any, depth int) any {
	if depth >= s.maxDepth {
		return v
	}
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if isSensitiveKey(k) {
				t[k] = SecretRedaction
				continue
			}
			if str, ok := val.(string); ok && isSensitiveValue(str) {
				t[k] = SecretRedaction
				continue
			}
			t[k] = s.sanitizeValue(val, depth+1)
		}
		return t
	case []any:
		for i, item := range t {
			t[i] = s.sanitizeValue(item, depth+1)
		}
		return t
	default:
		return v
	}
}

// isSensitiveKey reports whether a field name indicates secret-bearing data.
func isSensitiveKey(k string) bool {
	lk := strings.ToLower(k)
	for _, key := range []string{
		"token", "tokens", "secret", "secrets", "password", "passwd", "pwd",
		"apikey", "api_key", "apikey", "accesskey", "access_key", "secretkey",
		"secret_key", "privatekey", "private_key", "authorization", "auth",
		"cookie", "credentials", "credential", "clientsecret", "client_secret",
		"bearer", "data", // Kubernetes Secret `.data` is base64 payload
	} {
		if lk == key {
			return true
		}
	}
	return false
}

// isSensitiveValue reports whether a string looks like a credential.
func isSensitiveValue(s string) bool {
	if s == "" || len(s) < 6 {
		return false
	}
	for _, re := range sensitiveValuePatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// SanitizeSecretData redacts the opaque `data` map of a Kubernetes Secret
// object. It is exported so callers can scrub Secret objects specifically.
func SanitizeSecretData(data map[string]string) map[string]string {
	if data == nil {
		return nil
	}
	out := make(map[string]string, len(data))
	for k := range data {
		out[k] = SecretRedaction
	}
	return out
}
