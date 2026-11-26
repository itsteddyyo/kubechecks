package appdir

import (
	"testing"
	"testing/fstest"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestAddApp tests the AddAppSet method from the VcsToArgoMap type.
func TestAddApp(t *testing.T) {
	// Setup your mocks and expected calls here.

	v2a := NewVcsToArgoMap("vcs-username") // This would be mocked accordingly.
	app1 := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-1", Namespace: "default"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "https://github.com/argoproj/argo-cd.git",
				Path:    "test-app-1",
			},
		},
	}

	v2a.AddApp(app1)
	appDir := v2a.GetAppsInRepo("https://github.com/argoproj/argo-cd.git")

	assert.Equal(t, appDir.AppsCount(), 1)
	assert.Equal(t, len(appDir.appDirs["test-app-1"]), 1)

	// Assertions to verify the behavior here.
	app2 := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-2", Namespace: "default"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "https://github.com/argoproj/argo-cd.git",
				Path:    "test-app-2",
			},
		},
	}

	v2a.AddApp(app2)
	assert.Equal(t, appDir.AppsCount(), 2)
	assert.Equal(t, len(appDir.appDirs["test-app-2"]), 1)
}

func TestDeleteApp(t *testing.T) {
	// Setup your mocks and expected calls here.

	v2a := NewVcsToArgoMap("vcs-username") // This would be mocked accordingly.
	app1 := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-1", Namespace: "default"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "https://github.com/argoproj/argo-cd.git",
				Path:    "test-app-1",
			},
		},
	}
	// Assertions to verify the behavior here.
	app2 := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-2", Namespace: "default"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "https://github.com/argoproj/argo-cd.git",
				Path:    "test-app-2",
			},
		},
	}

	v2a.AddApp(app1)
	v2a.AddApp(app2)
	appDir := v2a.GetAppsInRepo("https://github.com/argoproj/argo-cd.git")

	assert.Equal(t, appDir.AppsCount(), 2)
	assert.Equal(t, len(appDir.appDirs["test-app-1"]), 1)
	assert.Equal(t, len(appDir.appDirs["test-app-2"]), 1)

	v2a.DeleteApp(app2)
	assert.Equal(t, appDir.AppsCount(), 1)
	assert.Equal(t, len(appDir.appDirs["test-app-2"]), 0)
}

func TestVcsToArgoMap_AddAppSet(t *testing.T) {
	type args struct {
		app *v1alpha1.ApplicationSet
	}
	tests := map[string]struct {
		name          string
		fields        VcsToArgoMap
		args          args
		expectedCount int
	}{
		"normal process, expect to get the appset stored in the map": {
			fields: NewVcsToArgoMap("dummyuser"),
			args: args{
				app: &v1alpha1.ApplicationSet{
					Spec: v1alpha1.ApplicationSetSpec{
						Template: v1alpha1.ApplicationSetTemplate{
							ApplicationSetTemplateMeta: v1alpha1.ApplicationSetTemplateMeta{},
							Spec: v1alpha1.ApplicationSpec{
								Source: &v1alpha1.ApplicationSource{
									RepoURL: "http://gitnotreal.local/unittest/",
									Path:    "apps/unittest/{{ values.cluster }}",
								},
							},
						},
					},
				},
			},
			expectedCount: 1,
		},
		"invalid appset": {
			fields: NewVcsToArgoMap("vcs-username"),
			args: args{
				app: &v1alpha1.ApplicationSet{
					Spec: v1alpha1.ApplicationSetSpec{
						Template: v1alpha1.ApplicationSetTemplate{
							ApplicationSetTemplateMeta: v1alpha1.ApplicationSetTemplateMeta{},
							Spec:                       v1alpha1.ApplicationSpec{},
						},
					},
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v2a := VcsToArgoMap{
				username:        tt.fields.username,
				appDirByRepo:    tt.fields.appDirByRepo,
				appSetDirByRepo: tt.fields.appSetDirByRepo,
			}
			v2a.AddAppSet(tt.args.app)
			assert.Equal(t, tt.expectedCount, len(v2a.appSetDirByRepo))
		})
	}
}

func TestVcsToArgoMap_DeleteAppSet(t *testing.T) {
	// Set up your mocks and expected calls here.

	v2a := NewVcsToArgoMap("vcs-username") // This would be mocked accordingly.
	app1 := &v1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-1", Namespace: "default"},
		Spec: v1alpha1.ApplicationSetSpec{
			Template: v1alpha1.ApplicationSetTemplate{
				Spec: v1alpha1.ApplicationSpec{
					Source: &v1alpha1.ApplicationSource{
						RepoURL: "https://github.com/argoproj/argo-cd.git",
						Path:    "test-app-1",
					},
				},
			},
		},
	}
	// Assertions to verify the behavior here.
	app2 := &v1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app-2", Namespace: "default"},
		Spec: v1alpha1.ApplicationSetSpec{
			Template: v1alpha1.ApplicationSetTemplate{
				Spec: v1alpha1.ApplicationSpec{
					Source: &v1alpha1.ApplicationSource{
						RepoURL: "https://github.com/argoproj/argo-cd.git",
						Path:    "test-app-2",
					},
				},
			},
		},
	}

	v2a.AddAppSet(app1)
	v2a.AddAppSet(app2)
	appDir := v2a.GetAppSetsInRepo("https://github.com/argoproj/argo-cd.git")

	assert.Equal(t, appDir.Count(), 2)
	assert.Equal(t, len(appDir.appSetDirs["test-app-1"]), 1)
	assert.Equal(t, len(appDir.appSetDirs["test-app-2"]), 1)

	v2a.DeleteAppSet(app2)
	assert.Equal(t, appDir.Count(), 1)
	assert.Equal(t, len(appDir.appSetDirs["test-app-2"]), 0)
}

// TestWalkKustomizeAppsSkipsSourcesWithoutADirectory covers the second root
// over-match: kustomize.ProcessKustomizationFile reports a directory holding no
// kustomization.yaml as a match rather than an error, so following a source with no
// path indexed the app at the repository root.
func TestWalkKustomizeAppsSkipsSourcesWithoutADirectory(t *testing.T) {
	v2a := NewVcsToArgoMap("vcs-username")
	app := newMultiSourceApp("headlamp-dev", testContentRepo, testContentRepo)
	v2a.AddApp(&app)

	// A repository with no kustomization.yaml anywhere, so every directory that is
	// looked up comes back as "the directory is what matters".
	rootFS := fstest.MapFS{
		"argocd/values.yaml": &fstest.MapFile{Data: []byte("{}")},
	}

	result := v2a.WalkKustomizeApps(testContentRepo, rootFS)

	assert.NotContains(t, result.appDirs, ".", "the repository root must not be indexed")
	assert.NotContains(t, result.appDirs, "", "the repository root must not be indexed")
	assert.Equal(t, map[string][]string{
		"charts/headlamp-config": {"headlamp-dev"},
	}, result.appDirs)

	// The union of the two directories is what the matcher actually consults, and an
	// unrelated change must not reach the app through either of them.
	combined := v2a.GetAppsInRepo(testContentRepo).Union(result)
	assert.Empty(t, combined.FindAppsBasedOnChangeList(
		[]string{"argocd/of-apps/infrastructure/infrastructure-3/templates/headlamp.yaml"}, "main"))
}
