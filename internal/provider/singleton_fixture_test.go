// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// The pipeline and aggregation resources are singletons — one per backend, with no
// name or ID that a test can vary. Every CI run in this repository shares a single
// backend, and nothing serializes the acceptance jobs across runs, so two runs that
// overlap in time write the same object. The run that loses the race then refreshes
// and sees a value it never wrote.
//
// The fixtures used to be byte-identical across runs (every run wrote "test-rule"),
// which made those collisions unattributable: a failing plan showed a value that
// could equally have come from another run, a leftover entry in the append-only
// store, or a human. Worse, two runs sitting on the same step clobbered each other
// invisibly — the intruding value matched the expected one exactly, so the test
// passed without having verified anything.
//
// Salting the fixtures with a per-run token fixes the diagnosis: an intruding value
// names its origin in the plan diff.
//
// This is deliberately not prevention. It makes collisions legible; it does not stop
// them, and by removing the identical-value coincidence it converts some silent
// passes into visible failures. Serializing the singleton acceptance groups across
// runs is the fix for the collisions themselves, and it belongs in CI configuration
// rather than here.
var testRunToken = sync.OnceValue(computeTestRunToken)

// computeTestRunToken derives a token that is stable for the lifetime of the test
// binary and traceable back to whatever produced it.
func computeTestRunToken() string {
	if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
		attempt := os.Getenv("GITHUB_RUN_ATTEMPT")
		if attempt == "" {
			attempt = "1"
		}
		// Run ID plus attempt is the granularity that matters: collisions are
		// between runs, and a re-run of the same run is a distinct writer.
		return sanitizeFixtureToken(fmt.Sprintf("gh%sa%s", runID, attempt))
	}

	// Outside CI, enough to tell two developers — or two shells — apart.
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	if host = sanitizeFixtureToken(host); len(host) > 12 {
		host = host[:12]
	}
	// The PID goes last so it survives the length cap, since it is the part that
	// distinguishes concurrent runs on one machine.
	return sanitizeFixtureToken(fmt.Sprintf("local%s%d", host, os.Getpid()))
}

// sanitizeFixtureToken reduces s to lowercase alphanumerics, so the result is safe to
// embed in a YAML scalar, a PromQL label matcher, and a regular expression alike, and
// caps its length to keep plan diffs readable.
func sanitizeFixtureToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}

	token := b.String()
	if token == "" {
		return "unknown"
	}

	const maxLen = 32
	if len(token) > maxLen {
		token = token[:maxLen]
	}
	return token
}

func TestSanitizeFixtureToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "already safe", input: "gh30796638259a1", expected: "gh30796638259a1"},
		{name: "uppercase is lowered", input: "GH123A1", expected: "gh123a1"},
		{name: "punctuation is dropped", input: "my-host.example_com", expected: "myhostexamplecom"},
		{name: "empty falls back", input: "", expected: "unknown"},
		{name: "all-unsafe falls back", input: "-._/", expected: "unknown"},
		{
			name:     "long input is capped",
			input:    strings.Repeat("a", 40),
			expected: strings.Repeat("a", 32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFixtureToken(tt.input); got != tt.expected {
				t.Errorf("sanitizeFixtureToken(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestComputeTestRunTokenUsesGitHubRunID(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "30796638259")
	t.Setenv("GITHUB_RUN_ATTEMPT", "2")

	if got, want := computeTestRunToken(), "gh30796638259a2"; got != want {
		t.Errorf("computeTestRunToken() = %q, want %q", got, want)
	}
}

func TestComputeTestRunTokenDefaultsAttempt(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "42")
	t.Setenv("GITHUB_RUN_ATTEMPT", "")

	if got, want := computeTestRunToken(), "gh42a1"; got != want {
		t.Errorf("computeTestRunToken() = %q, want %q", got, want)
	}
}

// fixtureTokenRegexp matches prefix followed by this run's token, for asserting that a
// singleton actually holds the document this run wrote. Without it the acceptance checks
// are satisfied by shape alone, so a value written by someone else passes.
func fixtureTokenRegexp(prefix string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(prefix + testRunToken()))
}

// The reformatted acceptance fixtures must stay semantically equal to the base ones,
// or TestAcc{Logs,Traces}PipelineResource_noDiffOnReformattedYaml would be asserting an
// empty plan for configs that genuinely differ — the test would pass for the wrong
// reason. Salting makes that easy to get wrong: a token in one fixture and not the
// other, or two different tokens, reads as a real content change.
func TestAccPipelineReformattedFixturesAreSemanticallyEqual(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		reformatted string
	}{
		{
			name:        "logs",
			base:        testAccLogsPipelineResourceConfig(),
			reformatted: testAccLogsPipelineResourceConfigReformatted(),
		},
		{
			name:        "traces",
			base:        testAccTracesPipelineResourceConfig(),
			reformatted: testAccTracesPipelineResourceConfigReformatted(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := extractHeredocYAML(t, tt.base)
			reformatted := extractHeredocYAML(t, tt.reformatted)

			if base == reformatted {
				t.Fatal("base and reformatted fixtures are byte-identical, so the acceptance test proves nothing")
			}
			if !YamlSemanticallyEqual(base, reformatted) {
				t.Errorf("fixtures are not semantically equal\nbase:\n%s\nreformatted:\n%s", base, reformatted)
			}

			for label, fixture := range map[string]string{"base": base, "reformatted": reformatted} {
				if !strings.Contains(fixture, testRunToken()) {
					t.Errorf("%s fixture does not carry the run token %q:\n%s", label, testRunToken(), fixture)
				}
			}
		})
	}
}

// The base-step token pattern must not also match the updated-step fixture, or the two
// steps would assert the same thing and an update that never landed would pass. The
// prefixes are nested by construction ("test-rule-" is a prefix of "test-rule-updated-"),
// so this only holds because the token follows immediately.
func TestFixtureTokenRegexpDiscriminatesBaseFromUpdated(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		updated string
		prefix  string
	}{
		{
			name:    "logs",
			base:    testAccLogsPipelineResourceConfig(),
			updated: testAccLogsPipelineResourceConfigUpdated(),
			prefix:  "test-rule-",
		},
		{
			name:    "traces",
			base:    testAccTracesPipelineResourceConfig(),
			updated: testAccTracesPipelineResourceConfigUpdated(),
			prefix:  "test-rule-",
		},
		{
			name:    "metrics aggregation",
			base:    testAccMetricsAggregationResourceConfig(),
			updated: testAccMetricsAggregationResourceConfigUpdated(),
			prefix:  "test_metric_counter_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := fixtureTokenRegexp(tt.prefix)
			if !re.MatchString(tt.base) {
				t.Errorf("%s does not match its own base fixture", re)
			}
			if re.MatchString(tt.updated) {
				t.Errorf("%s also matches the updated fixture, so the two steps do not discriminate", re)
			}
		})
	}
}

// extractHeredocYAML returns the body of the single <<-YAML heredoc in an HCL fixture.
func extractHeredocYAML(t *testing.T, config string) string {
	t.Helper()

	_, body, found := strings.Cut(config, "<<-YAML\n")
	if !found {
		t.Fatalf("fixture has no <<-YAML heredoc:\n%s", config)
	}

	body, _, found = strings.Cut(body, "\nYAML\n")
	if !found {
		t.Fatalf("fixture heredoc is unterminated:\n%s", config)
	}
	// Cut removed the whole terminator, including the newline that ended the last
	// line of YAML; put that one back.
	return body + "\n"
}

func TestComputeTestRunTokenOutsideCI(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "")

	token := computeTestRunToken()
	if !strings.HasPrefix(token, "local") {
		t.Errorf("computeTestRunToken() = %q, want a %q prefix outside CI", token, "local")
	}
	if token == sanitizeFixtureToken("") {
		t.Errorf("computeTestRunToken() fell back to the unknown placeholder")
	}
	if got := sanitizeFixtureToken(token); got != token {
		t.Errorf("computeTestRunToken() = %q is not already sanitized (%q)", token, got)
	}
}
