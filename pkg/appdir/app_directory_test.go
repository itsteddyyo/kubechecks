package appdir

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPathsAreJoinedProperly(t *testing.T) {
	rad := NewAppDirectory()
	app1 := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				Path: "/test1/test2",
				Helm: &v1alpha1.ApplicationSourceHelm{
					ValueFiles: []string{"one.yaml", "./two.yaml", "../three.yaml"},
					FileParameters: []v1alpha1.HelmFileParameter{
						{Name: "one", Path: "one.json"},
						{Name: "two", Path: "./two.json"},
						{Name: "three", Path: "../three.json"},
					},
				},
			},
		},
	}

	rad.AddApp(app1, "")

	assert.Equal(t, map[string]v1alpha1.Application{
		"test-app": app1,
	}, rad.appsMap)
	assert.Equal(t, map[string][]string{
		"/test1/test2": {"test-app"},
	}, rad.appDirs)
	assert.Equal(t, map[string][]string{
		"/test1/test2/one.json": {"test-app"},
		"/test1/test2/two.json": {"test-app"},
		"/test1/three.json":     {"test-app"},
		"/test1/test2/one.yaml": {"test-app"},
		"/test1/test2/two.yaml": {"test-app"},
		"/test1/three.yaml":     {"test-app"},
	}, rad.appFiles)
}

func TestShouldInclude(t *testing.T) {
	testcases := []struct {
		vcsMergeTarget  string
		argocdAppBranch string
		expected        bool
	}{
		{
			vcsMergeTarget:  "some-branch",
			argocdAppBranch: "some-branch",
			expected:        true,
		},
		{
			vcsMergeTarget:  "some-branch",
			argocdAppBranch: "some-other-branch",
			expected:        false,
		},
		{
			argocdAppBranch: "HEAD",
			vcsMergeTarget:  "main",
			expected:        true,
		},
		{
			argocdAppBranch: "HEAD",
			vcsMergeTarget:  "master",
			expected:        true,
		},
		{
			argocdAppBranch: "HEAD",
			vcsMergeTarget:  "other",
			expected:        false,
		},
		{
			argocdAppBranch: "",
			vcsMergeTarget:  "branch",
			expected:        true,
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%v", tc), func(t *testing.T) {
			actual := shouldInclude(v1alpha1.Application{
				Spec: v1alpha1.ApplicationSpec{
					Source: &v1alpha1.ApplicationSource{
						TargetRevision: tc.argocdAppBranch,
					},
				},
			}, tc.vcsMergeTarget)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestShouldIncludeMultiSourceApp(t *testing.T) {
	app := v1alpha1.Application{
		Spec: v1alpha1.ApplicationSpec{
			Sources: []v1alpha1.ApplicationSource{
				{TargetRevision: "3.10.0"},
				{TargetRevision: "refs/heads/main"},
			},
		},
	}

	assert.True(t, shouldInclude(app, "main"))
	assert.False(t, shouldInclude(app, "release"))

	app.Spec.Sources[1].TargetRevision = "refs/tags/main"
	assert.False(t, shouldInclude(app, "main"))
}

func TestDirContainsPath(t *testing.T) {
	testcases := []struct {
		name       string
		dir        string
		changePath string
		expected   bool
	}{
		{
			name:       "repo root path '.' matches any change",
			dir:        ".",
			changePath: "apps/production/app1.yaml",
			expected:   true,
		},
		{
			name:       "repo root path '/' matches any change",
			dir:        "/",
			changePath: "apps/production/app1.yaml",
			expected:   true,
		},
		{
			name:       "repo root path './' matches any change",
			dir:        "./",
			changePath: "apps/production/app1.yaml",
			expected:   true,
		},
		{
			name:       "empty path matches any change",
			dir:        "",
			changePath: "apps/production/app1.yaml",
			expected:   true,
		},
		{
			name:       "nested dir matches file within it",
			dir:        "charts/appchart",
			changePath: "charts/appchart/values.yaml",
			expected:   true,
		},
		{
			name:       "leading slash on dir is ignored",
			dir:        "/charts/appchart",
			changePath: "charts/appchart/values.yaml",
			expected:   true,
		},
		{
			name:       "exact dir match with no trailing file",
			dir:        "charts/appchart",
			changePath: "charts/appchart",
			expected:   true,
		},
		{
			name:       "sibling dir sharing a prefix does not match",
			dir:        "apps",
			changePath: "apps-other/foo.yaml",
			expected:   false,
		},
		{
			name:       "unrelated dir does not match",
			dir:        "charts/appchart",
			changePath: "apps/production/app1.yaml",
			expected:   false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, dirContainsPath(tc.dir, tc.changePath))
		})
	}
}

// TestFindAppsBasedOnChangeList_RootPath reproduces the root directory
// bug: an Application whose ArgoCD source path is the repo  root (".")
// must be matched by changes anywhere in the repo, not skipped.
func TestFindAppsBasedOnChangeList_RootPath(t *testing.T) {
	d := NewAppDirectory()
	app := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "example-application-production"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				Path: ".",
			},
		},
	}
	d.AddApp(app, "")

	changeList := []string{
		"charts/appchart/templates/_helpers.tpl",
		"apps/production/app1.yaml",
	}

	apps := d.FindAppsBasedOnChangeList(changeList, "")
	assert.Len(t, apps, 1)
	assert.Equal(t, "example-application-production", apps[0].Name)
}

// TestRemoveFromSlice performs tests on the removeFromSlice function.
func TestRemoveFromSlice(t *testing.T) {
	// Test for integers
	ints := []int{1, 2, 3, 4, 5}
	intsAfterRemoval := []int{1, 2, 4, 5}
	intsTest := func(t *testing.T) {
		result := removeFromSlice(ints, 3, func(a, b int) bool { return a == b })
		if !reflect.DeepEqual(result, intsAfterRemoval) {
			t.Errorf("Expected %v, got %v", intsAfterRemoval, result)
		}
	}

	// Test for strings
	strings := []string{"apple", "banana", "cherry", "date"}
	stringsAfterRemoval := []string{"apple", "cherry", "date"}
	stringsTest := func(t *testing.T) {
		result := removeFromSlice(strings, "banana", func(a, b string) bool { return a == b })
		if !reflect.DeepEqual(result, stringsAfterRemoval) {
			t.Errorf("Expected %v, got %v", stringsAfterRemoval, result)
		}
	}

	// Execute subtests
	t.Run("Integers", intsTest)
	t.Run("Strings", stringsTest)
	// Add more subtests for different generic types if necessary
}

const (
	testContentRepo = "https://gitlab.example.com/infrastructure/kubernetes.git"
	testValuesRepo  = "https://gitlab.example.com/infrastructure/shared-values.git"
	testChartRepo   = "https://kubernetes-sigs.github.io/headlamp"
)

// newMultiSourceApp builds an application shaped like a real multi-source ArgoCD app:
// a Helm chart from a chart repository, a content source in a git repository, and a
// `ref:` source whose root the content source's value files are written against.
func newMultiSourceApp(name, contentRepo, valuesRepo string) v1alpha1.Application {
	return v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.ApplicationSpec{
			Sources: []v1alpha1.ApplicationSource{
				{
					RepoURL:        testChartRepo,
					Chart:          "headlamp",
					TargetRevision: "0.45.0",
				},
				{
					RepoURL:        contentRepo,
					Path:           "charts/headlamp-config",
					TargetRevision: "refs/heads/main",
					Helm: &v1alpha1.ApplicationSourceHelm{
						ValueFiles: []string{
							"$values/argocd/values.yaml",
							"$values/argocd/values-dev.yaml",
						},
					},
				},
				{
					RepoURL:        valuesRepo,
					TargetRevision: "refs/heads/main",
					Ref:            "values",
				},
			},
		},
	}
}

func TestSourceDir(t *testing.T) {
	testcases := []struct {
		name     string
		src      v1alpha1.ApplicationSource
		expected string
		ok       bool
	}{
		{
			name: "a helm chart has no directory in the repository",
			src:  v1alpha1.ApplicationSource{RepoURL: testChartRepo, Chart: "headlamp"},
			ok:   false,
		},
		{
			name: "a ref source with no path has no directory",
			src:  v1alpha1.ApplicationSource{RepoURL: testValuesRepo, Ref: "values"},
			ok:   false,
		},
		{
			name:     "a ref source that also carries a path contributes it",
			src:      v1alpha1.ApplicationSource{RepoURL: testValuesRepo, Ref: "values", Path: "shared"},
			expected: "shared",
			ok:       true,
		},
		{
			name:     "a plain git source contributes its path",
			src:      v1alpha1.ApplicationSource{RepoURL: testContentRepo, Path: "charts/foo"},
			expected: "charts/foo",
			ok:       true,
		},
		{
			name:     "a plain git source with no path is genuinely the repository root",
			src:      v1alpha1.ApplicationSource{RepoURL: testContentRepo},
			expected: "",
			ok:       true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dir, ok := sourceDir(tc.src)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.expected, dir)
			}
		})
	}
}

func TestResolveValueFilePath(t *testing.T) {
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testValuesRepo)
	refs := sourceRefs(app)
	content := app.Spec.Sources[1]
	chart := app.Spec.Sources[0]

	testcases := []struct {
		name         string
		valueFile    string
		src          v1alpha1.ApplicationSource
		expectedRepo string
		expectedPath string
		ok           bool
	}{
		{
			name:         "a $ref path resolves against the ref source's repository and root",
			valueFile:    "$values/argocd/values.yaml",
			src:          content,
			expectedRepo: testValuesRepo,
			expectedPath: "argocd/values.yaml",
			ok:           true,
		},
		{
			name:         "a plain path resolves against the declaring source's directory",
			valueFile:    "values-override.yaml",
			src:          content,
			expectedRepo: testContentRepo,
			expectedPath: "charts/headlamp-config/values-override.yaml",
			ok:           true,
		},
		{
			name:         "a $ref path declared by a chart source still resolves",
			valueFile:    "$values/argocd/values.yaml",
			src:          chart,
			expectedRepo: testValuesRepo,
			expectedPath: "argocd/values.yaml",
			ok:           true,
		},
		{
			name:      "a plain path inside a chart is not a file in any watched repository",
			valueFile: "values.yaml",
			src:       chart,
			ok:        false,
		},
		{
			name:      "an unknown ref name resolves to nothing",
			valueFile: "$missing/argocd/values.yaml",
			src:       content,
			ok:        false,
		},
		{
			name:      "a $ref with no path resolves to nothing",
			valueFile: "$values",
			src:       content,
			ok:        false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			repoURL, filePath, ok := resolveValueFilePath(tc.valueFile, tc.src, refs)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.expectedRepo, repoURL)
				assert.Equal(t, tc.expectedPath, filePath)
			}
		})
	}
}

// TestAddAppMultiSourceSameRepo covers the common shape, where the content source and
// the ref source are the same repository.
func TestAddAppMultiSourceSameRepo(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testContentRepo)
	v2a.AddApp(&app)

	dir := v2a.GetAppsInRepo(testContentRepo)

	// The chart source and the ref source both carry an empty path, and neither may
	// leave the app registered at the repository root.
	assert.NotContains(t, dir.appDirs, "")
	assert.Equal(t, map[string][]string{
		"charts/headlamp-config": {"headlamp-dev"},
	}, dir.appDirs)

	// "$values/..." resolves against the ref source's root, not onto the content path.
	assert.Equal(t, map[string][]string{
		"argocd/values.yaml":     {"headlamp-dev"},
		"argocd/values-dev.yaml": {"headlamp-dev"},
	}, dir.appFiles)

	// A chart repository holds nothing the app depends on, so it stays empty.
	assert.Equal(t, 0, v2a.GetAppsInRepo(testChartRepo).AppsCount())
}

// TestAddAppMultiSourceCrossRepoRef covers a ref source in a different repository from
// the content source: each repository must index only its own paths.
func TestAddAppMultiSourceCrossRepoRef(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testValuesRepo)
	v2a.AddApp(&app)

	content := v2a.GetAppsInRepo(testContentRepo)
	assert.Equal(t, map[string][]string{
		"charts/headlamp-config": {"headlamp-dev"},
	}, content.appDirs)
	assert.Empty(t, content.appFiles, "the value files live in the values repository")

	values := v2a.GetAppsInRepo(testValuesRepo)
	assert.Empty(t, values.appDirs, "the content source's path must not leak into the values repository")
	assert.Equal(t, map[string][]string{
		"argocd/values.yaml":     {"headlamp-dev"},
		"argocd/values-dev.yaml": {"headlamp-dev"},
	}, values.appFiles)

	// A change to the shared values still triggers the app, from the values repository.
	assert.Len(t, values.FindAppsBasedOnChangeList([]string{"argocd/values.yaml"}, "main"), 1)
}

// TestMultiSourceAppMatchesOnlyItsOwnPaths is the regression test for the root
// over-match: an app whose sources include a chart and a ref must not be pulled into
// every check just because something, anywhere in the repository, changed.
func TestMultiSourceAppMatchesOnlyItsOwnPaths(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testContentRepo)
	v2a.AddApp(&app)

	dir := v2a.GetAppsInRepo(testContentRepo)

	assert.Empty(t,
		dir.FindAppsBasedOnChangeList([]string{"argocd/of-apps/bootstrap/templates/vault.yaml"}, "main"),
		"an unrelated change must not match")

	assert.Len(t,
		dir.FindAppsBasedOnChangeList([]string{"argocd/values.yaml"}, "main"), 1,
		"a change to a referenced value file must match")

	assert.Len(t,
		dir.FindAppsBasedOnChangeList([]string{"charts/headlamp-config/templates/deployment.yaml"}, "main"), 1,
		"a change to the app's own content must match")
}

// TestRemoveAppClearsEveryIndexedPath covers the leak where removal only ever cleaned
// up the first source's path.
func TestRemoveAppClearsEveryIndexedPath(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testValuesRepo)

	v2a.AddApp(&app)
	v2a.DeleteApp(&app)

	for _, repo := range []string{testContentRepo, testValuesRepo} {
		dir := v2a.GetAppsInRepo(repo)
		assert.Equal(t, 0, dir.AppsCount(), "repo %s still holds the app", repo)

		for indexed, names := range dir.appDirs {
			assert.Empty(t, names, "%s: appDirs[%q] still holds %v", repo, indexed, names)
		}
		for indexed, names := range dir.appFiles {
			assert.Empty(t, names, "%s: appFiles[%q] still holds %v", repo, indexed, names)
		}
	}
}

// TestAddAppSameNameInSeveralNamespaces covers applications in any namespace: the same
// application name in different namespaces identifies different applications, and each
// must be indexed under its own paths. Keying the index by name alone kept whichever
// one the informer happened to deliver first and dropped the others, so a change under
// a dropped application's path matched nothing.
func TestAddAppSameNameInSeveralNamespaces(t *testing.T) {
	paths := map[string]string{
		"bootstrap":      "argocd/of-apps/bootstrap/bootstrap-3/_charts/resources",
		"infrastructure": "argocd/of-apps/infrastructure/infrastructure-3/_charts/resources",
		"myodav":         "argocd/app/myodav/_charts/resources",
	}

	v2a := NewVcsToArgoMap("vcs-username")
	for namespace, sourcePath := range paths {
		app := v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Name: "resources-dev", Namespace: namespace},
			Spec: v1alpha1.ApplicationSpec{
				Sources: []v1alpha1.ApplicationSource{
					{RepoURL: testContentRepo, Path: sourcePath, TargetRevision: "refs/heads/main"},
					{RepoURL: testContentRepo, TargetRevision: "refs/heads/main", Ref: "values"},
				},
			},
		}
		v2a.AddApp(&app)
	}

	dir := v2a.GetAppsInRepo(testContentRepo)
	assert.Equal(t, len(paths), dir.AppsCount(), "every namespace's application must be indexed")

	// A change under one application's path matches that application and no other.
	for namespace, sourcePath := range paths {
		matched := dir.FindAppsBasedOnChangeList([]string{sourcePath + "/templates/configmap.yaml"}, "main")
		if !assert.Len(t, matched, 1, "a change under %s should match exactly one app", sourcePath) {
			continue
		}
		assert.Equal(t, "resources-dev", matched[0].Name)
		assert.Equal(t, namespace, matched[0].Namespace,
			"a change under %s must match the app in %s", sourcePath, namespace)
	}
}

// TestRemoveAppSameNameInSeveralNamespaces makes sure removing one of them leaves the
// others indexed.
func TestRemoveAppSameNameInSeveralNamespaces(t *testing.T) {
	newApp := func(namespace, sourcePath string) *v1alpha1.Application {
		return &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Name: "resources-dev", Namespace: namespace},
			Spec: v1alpha1.ApplicationSpec{
				Sources: []v1alpha1.ApplicationSource{
					{RepoURL: testContentRepo, Path: sourcePath, TargetRevision: "refs/heads/main"},
				},
			},
		}
	}

	bootstrap := newApp("bootstrap", "argocd/of-apps/bootstrap/bootstrap-3/_charts/resources")
	infrastructure := newApp("infrastructure", "argocd/of-apps/infrastructure/infrastructure-3/_charts/resources")

	v2a := NewVcsToArgoMap("vcs-username")
	v2a.AddApp(bootstrap)
	v2a.AddApp(infrastructure)
	v2a.DeleteApp(bootstrap)

	dir := v2a.GetAppsInRepo(testContentRepo)
	assert.Equal(t, 1, dir.AppsCount())

	matched := dir.FindAppsBasedOnChangeList(
		[]string{"argocd/of-apps/infrastructure/infrastructure-3/_charts/resources/templates/x.yaml"}, "main")
	if assert.Len(t, matched, 1, "the surviving app must still match") {
		assert.Equal(t, "infrastructure", matched[0].Namespace)
	}

	assert.Empty(t, dir.FindAppsBasedOnChangeList(
		[]string{"argocd/of-apps/bootstrap/bootstrap-3/_charts/resources/templates/x.yaml"}, "main"),
		"the removed app must no longer match")
}

// TestAddAppSetSameNameInSeveralNamespaces is the ApplicationSet equivalent.
func TestAddAppSetSameNameInSeveralNamespaces(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")

	for _, namespace := range []string{"bootstrap", "myodav"} {
		appSet := &v1alpha1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{Name: "tenants", Namespace: namespace},
			Spec: v1alpha1.ApplicationSetSpec{
				Template: v1alpha1.ApplicationSetTemplate{
					Spec: v1alpha1.ApplicationSpec{
						Source: &v1alpha1.ApplicationSource{
							RepoURL: testContentRepo,
							Path:    "appsets/" + namespace,
						},
					},
				},
			},
		}
		v2a.AddAppSet(appSet)
	}

	assert.Equal(t, 2, v2a.GetAppSetsInRepo(testContentRepo).Count(),
		"an appset name reused across namespaces must not collide")
}
