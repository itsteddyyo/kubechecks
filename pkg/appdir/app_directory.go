package appdir

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/rs/zerolog/log"

	"github.com/zapier/kubechecks/pkg"
)

// AppDirectory manages the mapping between applications and their associated directories and files.
// It provides functionality to track which applications are affected by changes in specific directories or files,
// and maintains a collection of Argo CD applications.
type AppDirectory struct {
	// appDirs maps directory paths to the names of applications that use those directories.
	// This is used to quickly identify which applications are affected when changes occur in a directory.
	appDirs map[string][]string

	// appFiles maps file paths to the names of applications that use those files.
	// This is used to quickly identify which applications are affected when specific files change.
	appFiles map[string][]string

	// appsMap stores the full Argo CD application definitions, indexed by application name.
	// This serves as the source of truth for application configurations.
	appsMap map[string]v1alpha1.Application
}

func NewAppDirectory() *AppDirectory {
	return &AppDirectory{
		appDirs:  make(map[string][]string),
		appFiles: make(map[string][]string),
		appsMap:  make(map[string]v1alpha1.Application),
	}
}

func (d *AppDirectory) AppsCount() int {
	return len(d.appsMap)
}

func (d *AppDirectory) AppFilesCount() int {
	return len(d.appFiles)
}

func (d *AppDirectory) AppDirsCount() int {
	return len(d.appDirs)
}

func (d *AppDirectory) Union(other *AppDirectory) *AppDirectory {
	var join AppDirectory
	join.appsMap = mergeMaps(d.appsMap, other.appsMap, takeFirst[v1alpha1.Application])
	join.appDirs = mergeMaps(d.appDirs, other.appDirs, mergeLists[string])
	join.appFiles = mergeMaps(d.appFiles, other.appFiles, mergeLists[string])
	return &join
}

// FindAppsBasedOnChangeList receives a list of modified file paths and
// returns the list of applications that are affected by the changes.
//
// changeList: a slice of strings representing the paths of modified files.
// targetBranch: the branch name to compare against the target revision of the applications.
// e.g. changeList = ["path/to/file1", "path/to/file2"]
func (d *AppDirectory) FindAppsBasedOnChangeList(changeList []string, targetBranch string) []v1alpha1.Application {
	log.Debug().Caller().Msgf("checking %d changes", len(changeList))

	appsSet := make(map[string]struct{})
	for _, changePath := range changeList {
		log.Debug().Caller().Msgf("change: %s", changePath)
		for dir, appKeys := range d.appDirs {
			if dirContainsPath(dir, changePath) {
				log.Debug().Caller().
					Str("changePath", changePath).
					Str("dir", dir).
					Msg("dir match!")
				for _, appKey := range appKeys {
					appsSet[appKey] = struct{}{}
				}
			}
		}

		appKeys, ok := d.appFiles[changePath]
		if ok {
			log.Debug().Caller().Str("changePath", changePath).Msg("file match!")
			for _, appKey := range appKeys {
				appsSet[appKey] = struct{}{}
			}
		}
	}

	var appsSlice []v1alpha1.Application
	for appKey := range appsSet {
		app, ok := d.appsMap[appKey]
		if !ok {
			log.Warn().Msgf("failed to find matched app named '%s'", appKey)
			continue
		}

		if !shouldInclude(app, targetBranch) {
			log.Debug().Caller().
				Msgf("target revision of %s is %s and does not match '%s'", appKey, getTargetRevision(app), targetBranch)
			continue
		}

		appsSlice = append(appsSlice, app)
	}

	log.Debug().Caller().Msgf("matched %d files into %d apps", len(changeList), len(appsSet))
	return appsSlice
}

// dirContainsPath reports whether changePath (a repo-relative file path, e.g.
// "charts/foo/values.yaml") falls within dir (an ArgoCD application source
// path, e.g. "charts/foo", "/charts/foo", ".", "/", or "").
//
// ArgoCD treats an empty path, ".", and "/" as all meaning "the repository
// root", so an app configured with any of those source paths is affected by
// every change in the repo. Beyond that, matching is done on path segment
// boundaries (rather than a raw string prefix) so that, for example, dir
// "apps" does not incorrectly match changePath "apps-other/foo.yaml".
func dirContainsPath(dir, changePath string) bool {
	changePath = path.Clean(changePath)

	dir = path.Clean(dir)
	if dir == "." || dir == "/" {
		// path.Clean("") == "."
		return true
	}
	dir = strings.TrimPrefix(dir, "/")

	if changePath == dir {
		return true
	}

	return strings.HasPrefix(changePath, dir+"/")
}

func getTargetRevision(app v1alpha1.Application) string {
	return app.Spec.GetSource().TargetRevision
}

func shouldInclude(app v1alpha1.Application, targetBranch string) bool {
	if app.Spec.SourceHydrator == nil && app.Spec.HasMultipleSources() {
		for _, source := range app.Spec.Sources {
			if targetRevisionMatches(source.TargetRevision, targetBranch) {
				return true
			}
		}
		return false
	}

	return targetRevisionMatches(getTargetRevision(app), targetBranch)
}

func targetRevisionMatches(targetRevision, targetBranch string) bool {
	if targetRevision == "" {
		return true
	}

	if pkg.AreSameTargetRefs(targetRevision, targetBranch) {
		return true
	}

	if targetRevision == "HEAD" {
		if targetBranch == "main" {
			return true
		}

		if targetBranch == "master" {
			return true
		}
	}

	return false
}

func (d *AppDirectory) GetApps(filter func(stub v1alpha1.Application) bool) []v1alpha1.Application {
	var result []v1alpha1.Application
	for _, value := range d.appsMap {
		if filter != nil && !filter(value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

// AddApp indexes the directories and files within repoURL whose modification should
// trigger a check of app.
//
// An application's sources may span repositories, and each repository gets its own
// AppDirectory. repoURL names the one this directory holds, so that only the paths
// that actually live in it are indexed here; passing an empty repoURL indexes every
// source, which is what a directory built outside VcsToArgoMap wants.
func (d *AppDirectory) AddApp(app v1alpha1.Application, repoURL string) {
	appKey := pkg.AppKey(app)

	if _, exists := d.appsMap[appKey]; exists {
		log.Debug().Caller().Msgf("app %s already exists", appKey)
		return
	}

	dirs, files := appIndexEntries(app, repoURL)
	if len(dirs) == 0 && len(files) == 0 {
		// Nothing in this repository can change what the app renders, so indexing it
		// here would only produce an entry that no change list can ever match.
		log.Debug().Caller().
			Str("appName", appKey).
			Str("repoURL", repoURL).
			Msg("app has no paths in this repository, not indexing it")
		return
	}

	log.Debug().
		Caller().
		Str("appName", appKey).
		Str("cluster-name", app.Spec.Destination.Name).
		Str("cluster-server", app.Spec.Destination.Server).
		Str("repoURL", repoURL).
		Strs("dirs", dirs).
		Strs("files", files).
		Msg("add app")

	d.appsMap[appKey] = app

	for _, dir := range dirs {
		d.addDir(appKey, dir)
	}

	for _, file := range files {
		d.addFile(appKey, file)
	}
}

// appIndexEntries returns the directories and files inside repoURL that app depends
// on, and is the single definition of that mapping so that adding and removing an
// app always touch exactly the same entries.
func appIndexEntries(app v1alpha1.Application, repoURL string) (dirs []string, files []string) {
	refs := sourceRefs(app)

	for _, src := range allSources(app) {
		if sourceBelongsToRepo(src.RepoURL, repoURL) {
			if dir, ok := sourceDir(src); ok {
				dirs = append(dirs, dir)
			}
		}

		helm := src.Helm
		if helm == nil {
			continue
		}

		// A value file may resolve into a repository other than the one declaring it,
		// so each is placed by where it resolves rather than by where it is written.
		valueFiles := make([]string, 0, len(helm.FileParameters)+len(helm.ValueFiles))
		for _, param := range helm.FileParameters {
			valueFiles = append(valueFiles, param.Path)
		}
		valueFiles = append(valueFiles, helm.ValueFiles...)

		for _, valueFile := range valueFiles {
			fileRepoURL, filePath, ok := resolveValueFilePath(valueFile, src, refs)
			if !ok || !sourceBelongsToRepo(fileRepoURL, repoURL) {
				continue
			}

			files = append(files, filePath)
		}
	}

	return dirs, files
}

// sourceDir returns the directory a source contributes to the change index, and
// whether it contributes one at all.
//
// Three kinds of source carry an empty Path and they do not mean the same thing. A
// Helm chart pulled from a chart repository has no git tree of its own, and a `ref:`
// source exists only to be named by "$<ref>/..." value file references. Neither is a
// directory in the repository, yet indexing them as one left the app registered at
// the repository root, where dirContainsPath matches every change anywhere in the
// repository. A plain git source with an empty path genuinely is the repository root
// and still matches everything, which is correct.
func sourceDir(src v1alpha1.ApplicationSource) (string, bool) {
	if src.Chart != "" {
		return "", false
	}

	if src.Ref != "" && src.Path == "" {
		return "", false
	}

	return src.Path, true
}

// sourceRefs indexes an application's `ref:` sources by their ref name, so that a
// "$<ref>/..." value file can be resolved back to the source it names.
func sourceRefs(app v1alpha1.Application) map[string]v1alpha1.ApplicationSource {
	refs := make(map[string]v1alpha1.ApplicationSource)

	for _, src := range allSources(app) {
		if src.Ref != "" {
			refs[src.Ref] = src
		}
	}

	return refs
}

// resolveValueFilePath resolves a Helm value file reference to the repository it
// lives in and its path within that repository.
//
// ArgoCD gives value files two forms. A plain relative path is taken from the
// directory of the source that declares it. A path beginning "$<ref>/" instead names
// the source whose `ref:` field is <ref>, and resolves against the root of THAT
// source, which may be an entirely different repository. Joining the second form onto
// the declaring source's path, as this once did, yielded paths such as
// "charts/foo/$values/shared/values.yaml", which no change list can ever match.
func resolveValueFilePath(
	valueFile string, src v1alpha1.ApplicationSource, refs map[string]v1alpha1.ApplicationSource,
) (repoURL, filePath string, ok bool) {
	if strings.HasPrefix(valueFile, "$") {
		refName, rest, found := strings.Cut(strings.TrimPrefix(valueFile, "$"), "/")
		if !found {
			return "", "", false
		}

		ref, found := refs[refName]
		if !found {
			return "", "", false
		}

		return ref.RepoURL, filepath.Join(ref.Path, rest), true
	}

	// A chart pulled from a chart repository has no git tree, so a value file inside
	// it is not a file any commit to a watched repository can change.
	if src.Chart != "" {
		return "", "", false
	}

	return src.RepoURL, filepath.Join(src.Path, valueFile), true
}

// sourceBelongsToRepo reports whether a source, or a resolved value file, lives in
// the repository an AppDirectory holds. An unset repository on either side imposes no
// restriction, which keeps a directory built directly, outside VcsToArgoMap, indexing
// everything it is given.
func sourceBelongsToRepo(srcRepoURL, repoURL string) bool {
	if srcRepoURL == "" || repoURL == "" {
		return true
	}

	return pkg.AreSameRepos(srcRepoURL, repoURL)
}

// allSources returns every source of an application, whether it declares one source
// or many.
func allSources(app v1alpha1.Application) []v1alpha1.ApplicationSource {
	if app.Spec.HasMultipleSources() {
		return app.Spec.Sources
	}

	if app.Spec.Source == nil {
		return nil
	}

	return []v1alpha1.ApplicationSource{*app.Spec.Source}
}

func (d *AppDirectory) addDir(appName, path string) {
	d.appDirs[path] = append(d.appDirs[path], appName)
}

func (d *AppDirectory) addFile(appName, path string) {
	d.appFiles[path] = append(d.appFiles[path], appName)
}

// RemoveApp takes app back out of the index for repoURL.
//
// It removes exactly the entries AddApp created, because both derive them from
// appIndexEntries. Removing only the first source's path, as this once did, left an
// app behind in every other directory and file it had been indexed under, so an
// update that changed a source path kept matching the old one forever.
func (d *AppDirectory) RemoveApp(app v1alpha1.Application, repoURL string) {
	appKey := pkg.AppKey(app)

	log.Debug().
		Caller().
		Str("appName", appKey).
		Str("cluster-name", app.Spec.Destination.Name).
		Str("cluster-server", app.Spec.Destination.Server).
		Str("repoURL", repoURL).
		Msg("delete app")

	// remove app from appsMap
	delete(d.appsMap, appKey)

	dirs, files := appIndexEntries(app, repoURL)

	for _, dir := range dirs {
		d.appDirs[dir] = removeFromSlice[string](d.appDirs[dir], appKey, func(a, b string) bool { return a == b })
	}

	for _, file := range files {
		d.appFiles[file] = removeFromSlice[string](d.appFiles[file], appKey, func(a, b string) bool { return a == b })
	}
}

func mergeMaps[T any](first map[string]T, second map[string]T, combine func(T, T) T) map[string]T {
	result := make(map[string]T)
	for key, value := range first {
		result[key] = value
	}
	for key, value := range second {
		exist, ok := result[key]
		if ok {
			result[key] = combine(exist, value)
		} else {
			result[key] = value
		}
	}
	return result
}

func mergeLists[T any](a []T, b []T) []T {
	return append(a, b...)
}

func takeFirst[T any](a, _ T) T {
	return a
}

func removeFromSlice[T any](slice []T, element T, equal func(T, T) bool) []T {
	for i, j := range slice {
		if equal(j, element) {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
