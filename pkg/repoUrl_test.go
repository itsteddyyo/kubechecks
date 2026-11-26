package pkg

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeStrings(t *testing.T) {
	type expected struct {
		RepoURL RepoURL
		Query   url.Values
	}
	testCases := []struct {
		input, name string
		expected    expected
	}{
		{
			name:  "simple github over ssh",
			input: "git@github.com:one/two",
			expected: expected{
				RepoURL: RepoURL{Host: "github.com", Path: "one/two"},
				Query:   make(url.Values),
			},
		},
		{
			name:  "simple github over https",
			input: "https://github.com/one/two",
			expected: expected{
				RepoURL: RepoURL{Host: "github.com", Path: "one/two"},
				Query:   make(url.Values),
			},
		},
		{
			name:  "simple gitlab over ssh",
			input: "git@gitlab.com:djeebus/helm-test.git",
			expected: expected{
				RepoURL: RepoURL{Host: "gitlab.com", Path: "djeebus/helm-test"},
				Query:   make(url.Values),
			},
		},
		{
			name:  "simple gitlab over https",
			input: "https://gitlab.com/djeebus/helm-test.git",
			expected: expected{
				RepoURL: RepoURL{Host: "gitlab.com", Path: "djeebus/helm-test"},
				Query:   make(url.Values),
			},
		},
		{
			name:  "simple gitlab over https with query",
			input: "https://gitlab.com/djeebus/helm-test.git?subdir=/blah",
			expected: expected{
				RepoURL: RepoURL{Host: "gitlab.com", Path: "djeebus/helm-test"},
				Query:   url.Values{"subdir": []string{"/blah"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("case %s", tc.input), func(t *testing.T) {
			repoURL, query, err := NormalizeRepoUrl(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected.RepoURL, repoURL)
			assert.Equal(t, tc.expected.Query, query)
		})
	}
}

func TestAreSameRepos(t *testing.T) {
	testcases := map[string]struct {
		input1, input2 string
		expected       bool
	}{
		"empty":                {"", "", true},
		"empty1":               {"", "blah", false},
		"empty2":               {"blah", "", false},
		"git-to-git":           {"git@github.com:zapier/kubechecks.git", "git@github.com:zapier/kubechecks.git", true},
		"no-git-suffix-to-git": {"git@github.com:zapier/kubechecks", "git@github.com:zapier/kubechecks.git", true},
		"https-to-git":         {"https://github.com/zapier/kubechecks", "git@github.com:zapier/kubechecks.git", true},
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			actual := AreSameRepos(tc.input1, tc.input2)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSameHost(t *testing.T) {
	const vcsBaseURL = "https://gitlab.int.odav.de"

	testcases := map[string]struct {
		input1, input2 string
		expected       bool
	}{
		"same self-hosted instance":     {"https://gitlab.int.odav.de/infrastructure/kubernetes.git", vcsBaseURL, true},
		"same instance over ssh":        {"git@gitlab.int.odav.de:infrastructure/kubernetes.git", vcsBaseURL, true},
		"same instance, differing case": {"https://GitLab.int.odav.de/infrastructure/kubernetes.git", vcsBaseURL, true},
		"public github source":          {"https://github.com/kairos-io/kairos-operator.git", vcsBaseURL, false},
		"helm chart repository":         {"https://kubernetes-sigs.github.io/headlamp", vcsBaseURL, false},
		"different repos, same host":    {"https://github.com/zapier/kubechecks", "https://github.com/argoproj/argo-cd", true},
		"local path has no host":        {"/tmp/kubechecks-repo-123", vcsBaseURL, false},
		"empty clone url":               {"", vcsBaseURL, false},
		"empty base url":                {"https://github.com/zapier/kubechecks", "", false},
		"both empty match nothing":      {"", "", false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, SameHost(tc.input1, tc.input2))
		})
	}
}

func TestIsConfiguredVcsHost(t *testing.T) {
	testcases := map[string]struct {
		cloneURL, vcsBaseURL string
		expected             bool
	}{
		"configured host":             {"https://gitlab.int.odav.de/infrastructure/kubernetes.git", "https://gitlab.int.odav.de", true},
		"foreign host":                {"https://github.com/kairos-io/kairos-operator.git", "https://gitlab.int.odav.de", false},
		"unset base url allows all":   {"https://github.com/kairos-io/kairos-operator.git", "", true},
		"unset base url, empty clone": {"", "", true},
		"local path against base url": {"/tmp/kubechecks-repo-123", "https://gitlab.int.odav.de", false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsConfiguredVcsHost(tc.cloneURL, tc.vcsBaseURL))
		})
	}
}
