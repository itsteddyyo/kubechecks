package pkg

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/chainguard-dev/git-urls"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type RepoURL struct {
	Host, Path string
}

func (r RepoURL) CloneURL(username string) string {
	if username != "" {
		return fmt.Sprintf("https://%s@%s/%s", username, r.Host, r.Path)
	}
	return fmt.Sprintf("https://%s/%s", r.Host, r.Path)
}

func NormalizeRepoUrl(s string) (RepoURL, url.Values, error) {
	var parser func(string) (*url.URL, error)

	if strings.HasPrefix(s, "http") {
		parser = url.Parse
	} else {
		parser = giturls.Parse
	}

	r, err := parser(s)
	if err != nil {
		return RepoURL{}, nil, err
	}

	r.Path = strings.TrimPrefix(r.Path, "/")
	r.Path = strings.TrimSuffix(r.Path, ".git")

	return RepoURL{
		Host: r.Host,
		Path: r.Path,
	}, r.Query(), nil
}

func Canonicalize(cloneURL string) (RepoURL, error) {
	parsed, _, err := NormalizeRepoUrl(cloneURL)
	if err != nil {
		return RepoURL{}, errors.Wrap(err, "failed to parse clone url")
	}

	return parsed, nil
}

// SameHost reports whether two git URLs live on the same host.
//
// It is the test for whether a repository belongs to the configured VCS instance, and
// therefore whether that instance's credentials may be sent to it. A URL whose host
// cannot be determined matches nothing, so callers gating credentials on this fail
// closed rather than disclosing a token to an unknown host.
func SameHost(url1, url2 string) bool {
	host1, ok1 := hostOf(url1)
	host2, ok2 := hostOf(url2)
	return ok1 && ok2 && host1 == host2
}

// IsConfiguredVcsHost reports whether a repository belongs to the configured VCS
// instance, and therefore whether that instance's credentials may be sent to it.
//
// An unset base URL imposes no restriction: the configuration loader always populates
// it, so an empty value means the caller constructed a config directly and should keep
// the previous behaviour rather than silently lose authentication.
func IsConfiguredVcsHost(cloneURL, vcsBaseURL string) bool {
	if vcsBaseURL == "" {
		return true
	}

	return SameHost(cloneURL, vcsBaseURL)
}

// hostOf returns the lowercased host of a git URL, reporting false when the URL cannot
// be parsed or carries no host - a local path, for instance.
func hostOf(rawURL string) (string, bool) {
	parsed, _, err := NormalizeRepoUrl(rawURL)
	if err != nil || parsed.Host == "" {
		return "", false
	}

	return strings.ToLower(parsed.Host), true
}

func AreSameRepos(url1, url2 string) bool {
	repo1, err := Canonicalize(url1)
	if err != nil {
		log.Warn().Msgf("failed to canonicalize %q", url1)
		return false
	}

	repo2, err := Canonicalize(url2)
	if err != nil {
		log.Warn().Msgf("failed to canonicalize %q", url2)
		return false
	}

	return repo1 == repo2
}
