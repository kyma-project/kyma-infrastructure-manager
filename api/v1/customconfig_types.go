package v1

import (
	registrycache "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CustomConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CustomConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

type CustomConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomConfig `json:"items"`
}

type CustomConfigSpec struct {
	RegistryCache []registrycache.RegistryCache `json:"cache"`
}
