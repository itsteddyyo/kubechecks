package pkg

import (
	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
)

// AppKey identifies an application by the namespace it lives in as well as its name.
//
// An application's name is unique only within its namespace. With applications in any
// namespace enabled, several namespaces routinely hold an application of the same
// name - a per-cluster "resources" application under bootstrap, under infrastructure
// and under a tenant namespace, say - so an index keyed by name alone keeps whichever
// one it sees first and silently discards the rest. Which one that is depends on the
// order the informer happens to deliver them in, so the same commit matches different
// applications from one restart to the next.
func AppKey(app v1alpha1.Application) string {
	return NamespacedName(app.Namespace, app.Name)
}

// AppSetKey identifies an application set the same way AppKey identifies an
// application.
func AppSetKey(appSet v1alpha1.ApplicationSet) string {
	return NamespacedName(appSet.Namespace, appSet.Name)
}

// NamespacedName joins a namespace and a name into the "namespace/name" form used to
// identify applications throughout kubechecks, and in the report it posts.
//
// A resource read from a rendered manifest may carry no namespace, in which case there
// is nothing to qualify with and the bare name is returned rather than a leading
// slash.
func NamespacedName(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "/" + name
}
