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
	"strings"
	"testing"
)

func TestSanitizeMapRedactsCredentials(t *testing.T) {
	s := NewSanitizer()
	payload := map[string]any{
		"name":        "my-app",
		"token":       "secret-jwt-token",
		"data":        map[string]any{"password": "hunter2"},
		"nested":      map[string]any{"api_key": "AKIA1234567890ABCDEF", "keep": "visible"},
		"safe":        "plain-value",
		"bearer":      "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nABC",
	}

	s.SanitizeAny(payload)

	assertRedacted(t, payload["token"])
	// `data` is the Kubernetes Secret payload key; the whole field is redacted.
	assertRedacted(t, payload["data"])
	assertRedacted(t, payload["nested"].(map[string]any)["api_key"])
	assertRedacted(t, payload["bearer"])
	assertRedacted(t, payload["private_key"])
	if payload["safe"] != "plain-value" {
		t.Errorf("safe value was modified: %v", payload["safe"])
	}
	if payload["name"] != "my-app" {
		t.Errorf("name was modified: %v", payload["name"])
	}
	if payload["nested"].(map[string]any)["keep"] != "visible" {
		t.Errorf("non-sensitive nested value was modified")
	}
}

func TestSanitizeJSON(t *testing.T) {
	s := NewSanitizer()
	in := []byte(`{"kind":"Secret","data":{"password":"hunter2"},"metadata":{"name":"creds"}}`)
	out := s.SanitizeJSON(in)

	if strings.Contains(string(out), "hunter2") {
		t.Errorf("secret value leaked in sanitized JSON: %s", out)
	}
	if !strings.Contains(string(out), SecretRedaction) {
		t.Errorf("expected redaction marker in output: %s", out)
	}
	// The document must remain valid JSON.
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("sanitized JSON is invalid: %v", err)
	}
}

func TestSanitizeJSONInvalidReturnsOriginal(t *testing.T) {
	s := NewSanitizer()
	raw := []byte("not json {")
	if got := s.SanitizeJSON(raw); string(got) != string(raw) {
		t.Errorf("invalid JSON should pass through unchanged")
	}
}

func TestSanitizeSecretData(t *testing.T) {
	in := map[string]string{"password": "hunter2", "key": "abc"}
	out := SanitizeSecretData(in)
	for k, v := range out {
		if v != SecretRedaction {
			t.Errorf("secret key %q not redacted: %q", k, v)
		}
	}
}

func TestMapStatusToCode(t *testing.T) {
	cases := []struct {
		status int
		want   int
	}{
		{401, ErrCodeAuthFailure},
		{403, ErrCodeAuthFailure},
		{400, ErrCodeInvalidParams},
		{422, ErrCodeInvalidParams},
		{500, ErrCodeUnreachable},
		{503, ErrCodeUnreachable},
		{200, ErrCodeUnknown},
	}
	for _, c := range cases {
		if got := mapStatusToCode(c.status); got != c.want {
			t.Errorf("mapStatusToCode(%d) = %d, want %d", c.status, got, c.want)
		}
	}
}

func TestAPIErrorFormatting(t *testing.T) {
	e := &APIError{Code: ErrCodeUnreachable, Op: "GET /api/pattern", Msg: "boom"}
	if !strings.Contains(e.Error(), "boom") {
		t.Errorf("error string missing message: %q", e.Error())
	}
	if e.Code != ErrCodeUnreachable {
		t.Errorf("code = %d", e.Code)
	}
}

func assertRedacted(t *testing.T, v any) {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("value is %T, want string", v)
	}
	if s != SecretRedaction {
		t.Errorf("value = %q, want %q", s, SecretRedaction)
	}
}
