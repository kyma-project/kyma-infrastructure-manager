package rtbootstrapper

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic/fake"
	clientgotesting "k8s.io/client-go/testing"
	"testing"
)

func TestManifestApplier_Apply_FromFile_ConfigMap(t *testing.T) {
	// given
	runtimeDynamicClientGetter := NewMockRuntimeDynamicClientGetter(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewSimpleDynamicClient(scheme)

	fakeDiscovery := &fakediscovery.FakeDiscovery{
		Fake:               &clientgotesting.Fake{},
		FakedServerVersion: nil,
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "configmaps",
					SingularName: "configmap",
					Namespaced:   true,
					Kind:         "ConfigMap",
					Verbs:        []string{"get", "list", "create", "update", "patch", "delete"},
				},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "deployments",
					SingularName: "deployment",
					Namespaced:   true,
					Kind:         "Deployment",
					Verbs:        []string{"get", "list", "create", "update", "patch", "delete"},
				},
			},
		},
	}

	runtime := minimalRuntime()
	runtimeDynamicClientGetter.EXPECT().Get(mock.Anything, runtime).Return(fakeClient, fakeDiscovery, nil)
	applier := NewManifestApplier("./testdata/manifests.yaml", runtimeDynamicClientGetter)

	// when
	err := applier.ApplyManifests(context.Background(), runtime)

	// then
	require.NoError(t, err)

	// verify ConfigMap created
	cmGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm, err := fakeClient.Resource(cmGVR).Namespace("default").Get(context.Background(), "testcm", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "ConfigMap", cm.GetKind())
	require.Equal(t, "testcm", cm.GetName())
	require.Equal(t, "default", cm.GetNamespace())

	// verify Deployment created
	deployGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	depl, err := fakeClient.Resource(deployGVR).Namespace("default").Get(context.Background(), "testdepl", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "Deployment", depl.GetKind())
	require.Equal(t, "testdepl", depl.GetName())
	require.Equal(t, "default", depl.GetNamespace())
}

func TestManifestApplier_Apply_FromFile_InvalidYAML(t *testing.T) {

}

func minimalRuntime() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
	}
}
