package rtbootstrapper

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"os"
)

type ManifestApplier struct {
	manifestsPath              string
	runtimeDynamicClientGetter RuntimeDynamicClientGetter
}

func NewManifestApplier(manifestsPath string, runtimeClientGetter RuntimeDynamicClientGetter) (*ManifestApplier, error) {
	return &ManifestApplier{
		manifestsPath:              manifestsPath,
		runtimeDynamicClientGetter: runtimeClientGetter,
	}, nil
}

func (ma ManifestApplier) ApplyManifests(ctx context.Context, runtime imv1.Runtime) error {
	f, err := os.Open(ma.manifestsPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

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
	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)

	for {
		u := &unstructured.Unstructured{}
		err = decoder.Decode(u)
		if err != nil {
			if err == io.EOF {
				break // processed all docs
			}
			return fmt.Errorf("decoding YAML: %w", err)
		}

		// Skip empty docs
		if u.GetKind() == "" {
			continue
		}

		if err := applyObject(ctx, dynamicClient, mapper, u, defaultNamespace); err != nil {
			return err
		}
	}

	return nil
}

func applyObject(
	ctx context.Context,
	dynClient dynamic.Interface,
	mapper meta.RESTMapper,
	obj *unstructured.Unstructured,
	defaultNamespace string,
) error {
	// Determine GVR and whether it's namespaced
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(schema.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}, gvk.Version)
	if err != nil {
		return fmt.Errorf("finding REST mapping for %s: %w", gvk.String(), err)
	}

	gvr := mapping.Resource

	// Decide namespace
	ns := obj.GetNamespace()
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if ns == "" {
			ns = defaultNamespace
			obj.SetNamespace(ns)
		}
	} else {
		// cluster-scoped, namespace must be empty
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

	// Try to get existing
	current, err := dr.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// If not found -> create
		if errors.IsNotFound(err) {
			fmt.Printf("Creating %s %s/%s\n", gvk.Kind, ns, name)
			_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
			return err
		}
		return fmt.Errorf("getting existing %s %s/%s: %w", gvk.Kind, ns, name, err)
	}

	// Keep resourceVersion so the Update succeeds
	obj.SetResourceVersion(current.GetResourceVersion())
	_, err = dr.Update(ctx, obj, metav1.UpdateOptions{})
	return err
}
