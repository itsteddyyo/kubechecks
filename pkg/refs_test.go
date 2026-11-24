package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeTargetRef(t *testing.T) {
	testcases := map[string]struct {
		input, expected string
	}{
		"bare branch":               {"main", "main"},
		"qualified branch":          {"refs/heads/main", "main"},
		"qualified nested branch":   {"refs/heads/feature/kubechecks", "feature/kubechecks"},
		"tag keeps its prefix":      {"refs/tags/customer", "refs/tags/customer"},
		"head":                      {"HEAD", "HEAD"},
		"commit sha":                {"8d7a2fdbbd4b9f1b4d1e0f4a1cf2b7b1e5c0a9d3", "8d7a2fdbbd4b9f1b4d1e0f4a1cf2b7b1e5c0a9d3"},
		"chart version":             {"3.1.1", "3.1.1"},
		"empty":                     {"", ""},
		"surrounding whitespace":    {"  refs/heads/main  ", "main"},
		"prefix only in the middle": {"release/refs/heads/main", "release/refs/heads/main"},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, NormalizeTargetRef(tc.input))
		})
	}
}

func TestAreSameTargetRefs(t *testing.T) {
	testcases := map[string]struct {
		ref1, ref2 string
		expected   bool
	}{
		"same bare branch":                    {"main", "main", true},
		"different bare branches":             {"main", "staging", false},
		"qualified matches bare":              {"refs/heads/main", "main", true},
		"bare matches qualified":              {"main", "refs/heads/main", true},
		"both qualified":                      {"refs/heads/main", "refs/heads/main", true},
		"qualified nested branch":             {"refs/heads/feature/kubechecks", "feature/kubechecks", true},
		"qualified branches differ":           {"refs/heads/main", "refs/heads/staging", false},
		"tag does not match same-name branch": {"refs/tags/customer", "customer", false},
		"tag matches itself":                  {"refs/tags/customer", "refs/tags/customer", true},
		"tag does not match branch":           {"refs/tags/customer", "refs/heads/customer", false},
		"head is not a branch name":           {"HEAD", "main", false},
		"chart version against branch":        {"3.1.1", "main", false},
		"empty against branch":                {"", "main", false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, AreSameTargetRefs(tc.ref1, tc.ref2))
		})
	}
}
