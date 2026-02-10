package rtbootstrapper

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"io"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type ManifestApplier struct {
	manifestsConfigMapName     string
	deploymentName             types.NamespacedName
	runtimeDynamicClientGetter RuntimeDynamicClientGetter
	runtimeClientGetter        RuntimeClientGetter
	kcpClient                  client.Client
}

func NewManifestApplier(manifestsConfigMapName string, deploymentName types.NamespacedName, runtimeClientGetter RuntimeClientGetter, runtimeDynamicClientGetter RuntimeDynamicClientGetter, kcpClient client.Client) *ManifestApplier {
	return &ManifestApplier{
		manifestsConfigMapName:     manifestsConfigMapName,
		runtimeDynamicClientGetter: runtimeDynamicClientGetter,
		runtimeClientGetter:        runtimeClientGetter,
		deploymentName:             deploymentName,
		kcpClient:                  kcpClient,
	}
}

func (ma ManifestApplier) ApplyManifests(ctx context.Context, runtime imv1.Runtime, manifests string) error {
	docs := strings.Split(manifests, "---")
	dynamicClient, discoveryClient, err := ma.runtimeDynamicClientGetter.Get(ctx, runtime)
	if err != nil {
		return fmt.Errorf("getting dynamic client: %w", err)
	}

	gr, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("getting API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	defaultNamespace := "default"

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		u := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		if err := decoder.Decode(u); err != nil && err != io.EOF {
			return fmt.Errorf("decoding YAML: %w", err)
		}
		if u.GetKind() == "" {
			continue
		}
		if err := applyObject(ctx, dynamicClient, mapper, u, defaultNamespace); err != nil {
			return err
		}
	}
	return nil
}

func (ma ManifestApplier) getManifests(ctx context.Context, kcpClient client.Client) (string, error) {
	var manifestsConfigMap corev1.ConfigMap

	err := kcpClient.Get(ctx, client.ObjectKey{Name: ma.manifestsConfigMapName, Namespace: "kcp-system"}, &manifestsConfigMap)

	if err != nil {
		return "", fmt.Errorf("getting ConfigMap with manifests: %w", err)
	}

	return manifestsConfigMap.Data["manifests.yaml"], nil
}

func applyObject(
	ctx context.Context,
	dynClient dynamic.Interface,
	mapper meta.RESTMapper,
	obj *unstructured.Unstructured,
	defaultNamespace string,
) error {
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(schema.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}, gvk.Version)
	if err != nil {
		return fmt.Errorf("finding REST mapping for %s: %w", gvk.String(), err)
	}

	gvr := mapping.Resource

	ns := obj.GetNamespace()
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if ns == "" {
			ns = defaultNamespace
			obj.SetNamespace(ns)
		}
	} else {
		ns = ""
		obj.SetNamespace("")
	}

	name := obj.GetName()
	if name == "" {
		return fmt.Errorf("object %s is missing metadata.name", gvk.String())
	}

	var dr dynamic.ResourceInterface
	if ns == "" {
		dr = dynClient.Resource(gvr)
	} else {
		dr = dynClient.Resource(gvr).Namespace(ns)
	}

	current, err := dr.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
			return err
		}
		return fmt.Errorf("getting existing %s %s/%s: %w", gvk.Kind, ns, name, err)
	}

	obj.SetResourceVersion(current.GetResourceVersion())

	patchBytes, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshaling object to JSON: %w", err)
	}

	_, err = dr.Patch(ctx, name, types.ApplyPatchType, patchBytes, metav1.PatchOptions{
		Force:        ptr.To(true),
		FieldManager: fieldManagerName,
	})

	return err
}

func (ma ManifestApplier) Status(ctx context.Context, runtime imv1.Runtime) (InstallationStatus, string, error) {
	var deployment v1.Deployment
	manifestsToInstall, err := ma.getManifests(ctx, ma.kcpClient)
	if err != nil {
		return StatusFailed, "", fmt.Errorf("getting manifests: %w", err)
	}

	runtimeClient, err := ma.runtimeClientGetter.Get(ctx, runtime)
	if err != nil {
		return StatusFailed, "", fmt.Errorf("getting runtime client: %w", err)
	}

	err = runtimeClient.Get(ctx, ma.deploymentName, &deployment)
	if err != nil && errors.IsNotFound(err) {
		return StatusNotStarted, manifestsToInstall, nil
	}

	if err != nil {
		return StatusFailed, "", fmt.Errorf("getting deployment: %w", err)
	}

	if isDeploymentReady(&deployment) {
		upgradeNeeded, err := isDeploymentToBeUpdated(&deployment, manifestsToInstall)
		if err != nil {
			return StatusFailed, "", fmt.Errorf("checking if deployment needs update: %w", err)
		}

		if upgradeNeeded {
			return StatusUpgradeNeeded, manifestsToInstall, nil
		}

		return StatusReady, manifestsToInstall, nil
	}

	if isDeploymentProgressing(&deployment) {
		return StatusInProgress, manifestsToInstall, nil
	}

	// When we got here the timeout occurred
	return StatusFailed, "", nil
}

func isDeploymentReady(dep *v1.Deployment) bool {
	if dep.Status.ReadyReplicas < *dep.Spec.Replicas {
		return false
	}

	available := false

	for _, cond := range dep.Status.Conditions {
		switch cond.Type {
		case v1.DeploymentAvailable:
			if cond.Status == "True" {
				available = true
			}
		}
	}

	return available
}

func isDeploymentToBeUpdated(dep *v1.Deployment, manifestsPath string) (bool, error) {
	deploymentToBeApplied, err := getDeploymentToBeApplied(manifestsPath)

	if err != nil {
		return false, fmt.Errorf("getting deployment to be applied: %w", err)
	}

	versionToBeApplied := deploymentToBeApplied.GetLabels()["app.kubernetes.io/version"]
	currentVersion := dep.GetLabels()["app.kubernetes.io/version"]

	return versionToBeApplied != currentVersion, nil
}

func getDeploymentToBeApplied(manifests string) (*unstructured.Unstructured, error) {
	docs := strings.Split(manifests, "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		u := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		if err := decoder.Decode(u); err != nil && err != io.EOF {
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}
		if u.GetKind() == "" {
			continue
		}

		if u.GetKind() == "Deployment" {
			return u, nil
		}
	}

	return nil, nil
}

func isDeploymentProgressing(dep *v1.Deployment) bool {
	progressing := false

	for _, cond := range dep.Status.Conditions {
		switch cond.Type {
		case v1.DeploymentProgressing:
			if cond.Status == "True" {
				progressing = true
			}
		}
	}

	return progressing
}
