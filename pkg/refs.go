package pkg

import "strings"

// BranchRefPrefix is the prefix git uses for fully qualified branch references.
const BranchRefPrefix = "refs/heads/"

// NormalizeTargetRef reduces a git reference to the form used for comparisons.
//
// ArgoCD accepts a fully qualified branch reference ("refs/heads/main") wherever it
// accepts a bare branch name ("main"), while VCS providers only ever report bare branch
// names in a pull request's base and head refs. Comparing the two forms literally makes
// an Application that tracks "refs/heads/main" look like it tracks an unrelated branch.
//
// Only the branch prefix is stripped. Tags keep their "refs/tags/" prefix, because a tag
// and a branch that share a name are different references and must not compare equal.
// Anything else - a bare branch name, a tag, a commit SHA, "HEAD", or a Helm chart
// version - is returned unchanged.
func NormalizeTargetRef(ref string) string {
	return strings.TrimPrefix(strings.TrimSpace(ref), BranchRefPrefix)
}

// AreSameTargetRefs reports whether two git references point at the same branch, tag or
// revision, ignoring whether either side spells a branch in its fully qualified form.
func AreSameTargetRefs(ref1, ref2 string) bool {
	return NormalizeTargetRef(ref1) == NormalizeTargetRef(ref2)
}
