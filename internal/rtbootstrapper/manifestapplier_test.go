package rtbootstrapper

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic/fake"
	clientgotesting "k8s.io/client-go/testing"
	"testing"
)

func TestManifestApplier_Apply_FromFile_ConfigMap(t *testing.T) {
	// given
	runtime := minimalRuntime()
	runtimeDynamicClientGetter := NewMockRuntimeDynamicClientGetter(t)
	fakeClient := &fake.FakeDynamicClient{}

	runtimeDynamicClientGetter.EXPECT().Get(mock.Anything, runtime).Return(fakeClient, &fakediscovery.FakeDiscovery{
		Fake:               &clientgotesting.Fake{},
		FakedServerVersion: nil,
	}, nil)
	applier := NewManifestApplier("./test/manifests.yaml", runtimeDynamicClientGetter)

	// when
	err := applier.ApplyManifests(context.Background(), runtime)
	require.NoError(t, err)
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
