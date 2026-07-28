// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

// Shared harness for the pipeline singleton tests. Unlike the resource-level tests, these
// drive the provider through the real Terraform CLI, which is what makes them able to show
// whether Terraform considers a plan empty. They skip under -short and when the terraform
// binary is unavailable.

// Values shared by the logs and traces pipeline tests; the two resources have identical
// schemas, so the same fixtures serve both.
const (
	testPipelineStateValue = "ottlRules:\n- ruleName: test-rule\n  conditions:\n    - container_name == \"nginx\"\n"
	// Same document as testPipelineStateValue, serialized differently.
	testPipelineRemoteValue = "ottlRules:\n  - conditions:\n      - container_name == \"nginx\"\n    ruleName: test-rule\n"
	testPipelineOtherValue  = "ottlRules:\n- ruleName: other-rule\n"

	testPipelineStateTimestamp  = "2026-07-01T10:00:00.000Z"
	testPipelineRemoteTimestamp = "2026-07-27T09:00:00.000Z"
)

type fakePipelineBackend struct {
	t        *testing.T
	endpoint string

	// mu guards the fields below, written by handler goroutines and read by the test.
	mu      sync.Mutex
	stored  string
	present bool
	writes  int

	// remoteFn transforms the stored value before returning it, standing in for the
	// backend's own serialization. Nil means "return it verbatim".
	remoteFn func(string) string

	// bumpTimestampOnRead reports a later timestamp on every read.
	bumpTimestampOnRead bool
	readBumps           int
}

func (f *fakePipelineBackend) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(f.endpoint, func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			if !f.present {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if f.bumpTimestampOnRead {
				f.readBumps++
			}
			f.respond(w, http.StatusOK)
		case http.MethodPost, http.MethodPut:
			var body struct{ Value string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				f.t.Errorf("could not decode %s body: %v", r.Method, err)
			}
			f.stored = body.Value
			f.present = true
			f.writes++
			status := http.StatusOK
			if r.Method == http.MethodPost {
				status = http.StatusCreated
			}
			f.respond(w, status)
		case http.MethodDelete:
			f.present = false
			f.stored = ""
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"deleted"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func (f *fakePipelineBackend) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

// respond must be called with f.mu held.
func (f *fakePipelineBackend) respond(w http.ResponseWriter, status int) {
	value := f.stored
	if f.remoteFn != nil {
		value = f.remoteFn(f.stored)
	}

	// Writes advance the timestamp; reads advance it too when bumpTimestampOnRead is set.
	ts := fmt.Sprintf("2026-07-27T09:%02d:00.000Z", f.writes)
	if f.bumpTimestampOnRead && f.readBumps > 0 {
		ts = fmt.Sprintf("2026-07-27T10:%02d:00.000Z", f.readBumps)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"uuid":              "6b09aee0-9001-4095-afb1-152b487e5bdd",
		"value":             value,
		"created_timestamp": ts,
		"created_by":        "terraform",
	})
}

// reserializeYaml round-trips a document through a YAML library, the way the config store
// does: same meaning, different bytes.
func reserializeYaml(stored string) string {
	if stored == "" {
		return ""
	}
	var data interface{}
	if err := yaml.Unmarshal([]byte(stored), &data); err != nil {
		return stored
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return stored
	}
	return string(out)
}

// startFakePipelineBackend returns the backend and a provider block pointing at it. The
// block sets api_url and api_key inline, and inline provider config wins over the
// GROUNDCOVER_* environment variables, so these tests cannot reach a real backend.
//
// Driving the real Terraform CLI puts these outside what -short covers, and the binary is
// not always present, so both cases skip rather than fail.
func startFakePipelineBackend(t *testing.T, backend *fakePipelineBackend) (*fakePipelineBackend, string) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping Terraform CLI-driven test in -short mode")
	}
	if _, err := exec.LookPath("terraform"); err != nil && os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		t.Skip("terraform CLI not found in PATH; skipping Terraform CLI-driven test")
	}

	if backend.endpoint == "" {
		t.Fatal("fakePipelineBackend.endpoint must be set")
	}
	backend.t = t
	srv := httptest.NewServer(backend.handler())
	t.Cleanup(srv.Close)

	return backend, `
provider "groundcover" {
  api_key    = "fake-api-key"
  backend_id = "fake-backend"
  api_url    = "` + srv.URL + `"
}
`
}
