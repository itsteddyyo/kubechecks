package pkg

import (
	"testing"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespacedName(t *testing.T) {
	testcases := []struct {
		name      string
		namespace string
		appName   string
		expected  string
	}{
		{
			name:      "a namespaced application is qualified",
			namespace: "infrastructure",
			appName:   "resources-dev",
			expected:  "infrastructure/resources-dev",
		},
		{
			name:      "the same name in another namespace is a different key",
			namespace: "bootstrap",
			appName:   "resources-dev",
			expected:  "bootstrap/resources-dev",
		},
		{
			name:     "a resource with no namespace keeps its bare name",
			appName:  "resources-dev",
			expected: "resources-dev",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, NamespacedName(tc.namespace, tc.appName))
		})
	}
}

func TestAppKey(t *testing.T) {
	bootstrap := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "resources-dev", Namespace: "bootstrap"},
	}
	infrastructure := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "resources-dev", Namespace: "infrastructure"},
	}

	assert.Equal(t, "bootstrap/resources-dev", AppKey(bootstrap))
	assert.Equal(t, "infrastructure/resources-dev", AppKey(infrastructure))
	assert.NotEqual(t, AppKey(bootstrap), AppKey(infrastructure),
		"applications sharing a name in different namespaces must not share a key")
}

func TestAppSetKey(t *testing.T) {
	appSet := v1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{Name: "tenants", Namespace: "myodav"},
	}

	assert.Equal(t, "myodav/tenants", AppSetKey(appSet))
}
