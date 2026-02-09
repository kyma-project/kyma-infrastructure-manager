package rtbootstrapper

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic/fake"
	clientgotesting "k8s.io/client-go/testing"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManifestApplier_Apply_FromFile(t *testing.T) {
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

	rt := minimalRuntime()
	runtimeDynamicClientGetter.EXPECT().Get(mock.Anything, rt).Return(fakeClient, fakeDiscovery, nil)
	applier := NewManifestApplier("./testdata/manifests.yaml", types.NamespacedName{}, "", nil, runtimeDynamicClientGetter)

	// when
	err := applier.ApplyManifests(context.Background(), rt)

	// then
	require.NoError(t, err)

	cmGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm, err := fakeClient.Resource(cmGVR).Namespace("default").Get(context.Background(), "testcm", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "ConfigMap", cm.GetKind())
	require.Equal(t, "testcm", cm.GetName())
	require.Equal(t, "default", cm.GetNamespace())

	deployGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	depl, err := fakeClient.Resource(deployGVR).Namespace("default").Get(context.Background(), "testdepl", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "Deployment", depl.GetKind())
	require.Equal(t, "testdepl", depl.GetName())
	require.Equal(t, "default", depl.GetNamespace())
}

func TestManifestApplier_ManifestErrors(t *testing.T) {
	// given
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewSimpleDynamicClient(scheme)

	fakeDiscovery := &fakediscovery.FakeDiscovery{
		Fake:               &clientgotesting.Fake{},
		FakedServerVersion: nil,
	}

	runtimeDynamicClientGetter := NewMockRuntimeDynamicClientGetter(t)
	runtimeDynamicClientGetter.EXPECT().Get(mock.Anything, mock.Anything).Return(fakeClient, fakeDiscovery, nil)

	t.Run("Failed to decode file", func(t *testing.T) {
		//when
		applier := NewManifestApplier("./testdata/invalid.yaml", types.NamespacedName{}, "", nil, runtimeDynamicClientGetter)
		err := applier.ApplyManifests(context.Background(), minimalRuntime())

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "decoding YAML")
	})

	t.Run("Failed to open file", func(t *testing.T) {
		// when
		applier := NewManifestApplier("nonexistent", types.NamespacedName{}, "", nil, runtimeDynamicClientGetter)
		err := applier.ApplyManifests(context.Background(), minimalRuntime())

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestManifestApplier_Status(t *testing.T) {
	// given
	runtimeClientGetter := NewMockRuntimeClientGetter(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	readyDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready-depl",
			Namespace: "default",
			Labels: map[string]string{
				"app":                       "ready",
				"app.kubernetes.io/version": "1.0.1"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(3)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "europe-docker.pkg.dev/kyma-project/prod/rt-bootstrapper:1.0.1",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      3,
			ReadyReplicas: 3,
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue, Reason: "MinimumReplicasAvailable"},
			},
		},
	}

	upgradeDepl := readyDepl.DeepCopy()
	upgradeDepl.Name = "upgrade-depl"
	upgradeDepl.ObjectMeta.Labels["app.kubernetes.io/version"] = "1.0.0"

	inProgressDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "progress-depl",
			Namespace: "default",
			Labels:    map[string]string{"app": "progress"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(3)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "europe-docker.pkg.dev/kyma-project/prod/rt-bootstrapper:1.0.1",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      3,
			ReadyReplicas: 0,
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "ReplicaSetUpdated"},
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
			},
		},
	}

	failedDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-depl",
			Namespace: "default",
			Labels:    map[string]string{"app": "failed"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(3)),
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 0,
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse, Reason: "FailedCreate"},
			},
		},
	}

	fakeClient := ctrlclientfake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(readyDepl, upgradeDepl, inProgressDepl, failedDepl).
		Build()

	rt := minimalRuntime()
	runtimeClientGetter.EXPECT().Get(mock.Anything, rt).Return(fakeClient, nil)

	ctx := context.Background()

	t.Run("StatusReady", func(t *testing.T) {
		// when

		applier := NewManifestApplier("./testdata/manifests.yaml", types.NamespacedName{Name: "ready-depl", Namespace: "default"}, "1.0.0", runtimeClientGetter, nil)
		status, err := applier.Status(ctx, rt)

		//then
		require.NoError(t, err)
		require.Equal(t, StatusReady, status)
	})

	t.Run("StatusProgressing", func(t *testing.T) {
		// when
		applier := NewManifestApplier("", types.NamespacedName{Name: "progress-depl", Namespace: "default"}, "1.0.0", runtimeClientGetter, nil)
		status, err := applier.Status(ctx, rt)

		// then
		require.NoError(t, err)
		require.Equal(t, StatusInProgress, status)
	})

	t.Run("StatusFailed", func(t *testing.T) {
		// when
		applier := NewManifestApplier("", types.NamespacedName{Name: "failed-depl", Namespace: "default"}, "1.0.0", runtimeClientGetter, nil)
		status, err := applier.Status(ctx, rt)

		// then
		require.NoError(t, err)
		require.Equal(t, StatusFailed, status)
	})

	t.Run("StatusNotStarted", func(t *testing.T) {
		// when
		applier := NewManifestApplier("", types.NamespacedName{Name: "missing-depl", Namespace: "default"}, "1.0.0", runtimeClientGetter, nil)
		status, err := applier.Status(ctx, rt)

		// then
		require.NoError(t, err)
		require.Equal(t, StatusNotStarted, status)
	})

	t.Run("StatusUpgradeNeeded", func(t *testing.T) {
		// when
		applier := NewManifestApplier("./testdata/manifests.yaml", types.NamespacedName{Name: "upgrade-depl", Namespace: "default"}, "1.0.1", runtimeClientGetter, nil)
		status, err := applier.Status(ctx, rt)

		//then
		require.NoError(t, err)
		require.Equal(t, StatusUpgradeNeeded, status)
	})
}

func TestManifestApplier_StatusErrors(t *testing.T) {
	ctx := context.Background()
	rt := minimalRuntime()

	t.Run("Failed to get client", func(t *testing.T) {
		runtimeClientGetter := NewMockRuntimeClientGetter(t)
		runtimeClientGetter.EXPECT().Get(mock.Anything, rt).Return(nil, errors.New("failed"))

		applier := NewManifestApplier("", types.NamespacedName{Name: "depl", Namespace: "default"}, "", runtimeClientGetter, nil)
		_, err := applier.Status(ctx, rt)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})

	t.Run("Failed to get deployment", func(t *testing.T) {
		fakeClient := ctrlclientfake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("get error")
			},
		}).Build()

		runtimeClientGetter := NewMockRuntimeClientGetter(t)
		runtimeClientGetter.EXPECT().Get(mock.Anything, rt).Return(fakeClient, nil)
		applier := NewManifestApplier("", types.NamespacedName{Name: "depl", Namespace: "default"}, "", runtimeClientGetter, nil)
		_, err := applier.Status(ctx, rt)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get error")
	})
}

func minimalRuntime() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
	}
}
